package store

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/davydany/ClawIDE/internal/model"
)

// Task-ID is embedded as an HTML comment on the H3 line so it survives rendering and round-trip.
var taskIDPattern = regexp.MustCompile(`<!--\s*id:\s*([^\s>]+)\s*-->`)

// Linked branch is embedded as a second HTML comment on the H3 line, parallel to the id comment.
// Branch names can contain "/" (e.g. feature/foo) but not whitespace or ">" per git's rules.
var taskBranchPattern = regexp.MustCompile(`<!--\s*branch:\s*([^\s>]+)\s*-->`)

// commentHeaderPattern matches the first line of a comment block:
//
//	**[2026-04-15 10:22 · AI/claude/sonnet]** Body text...
//
// Group 1 = timestamp string, group 2 = author string, group 3 = first-line body.
var commentHeaderPattern = regexp.MustCompile(`^\*\*\[(.+?)\s+·\s+(.+?)\]\*\*\s?(.*)$`)

// ParseBoard parses a tasks.md byte slice into a Board. Returns a Board even on partial failures;
// the parser is forgiving of hand-edits. Unknown content above the first H1 is preserved in
// Board.Preamble. Missing task IDs are left empty so the caller controls when IDs are minted.
func ParseBoard(data []byte) (model.Board, error) {
	var board model.Board
	lines := strings.Split(string(data), "\n")
	// Trim a trailing empty element caused by the final "\n" so we don't emit a phantom blank line.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var (
		preambleLines []string
		sawFirstH1    bool
		curColumnIdx  = -1
		curGroupIdx   = -1 // index within current column's Groups slice
		curTaskIdx    = -1 // index within current group's Tasks slice
		// descBuf accumulates non-comment prose for the current task.
		descBuf []string
		// commentBuf accumulates lines of the comment currently being built.
		commentBuf []string
		inComment  bool
	)

	// flushComment materializes commentBuf into a Comment on the current task and resets the buffer.
	flushComment := func() {
		if !inComment || len(commentBuf) == 0 {
			inComment = false
			commentBuf = nil
			return
		}
		c := parseComment(commentBuf)
		if curTaskIdx >= 0 {
			task := &board.Columns[curColumnIdx].Groups[curGroupIdx].Tasks[curTaskIdx]
			task.Comments = append(task.Comments, c)
		}
		commentBuf = nil
		inComment = false
	}

	// flushDescription commits the accumulated description lines to the current task.
	flushDescription := func() {
		if curTaskIdx < 0 || len(descBuf) == 0 {
			descBuf = nil
			return
		}
		desc := strings.TrimSpace(strings.Join(descBuf, "\n"))
		task := &board.Columns[curColumnIdx].Groups[curGroupIdx].Tasks[curTaskIdx]
		// If we already have a description (because the user wrote prose between comments),
		// append with a blank line so nothing is lost on round-trip.
		if task.Description != "" && desc != "" {
			task.Description = task.Description + "\n\n" + desc
		} else if desc != "" {
			task.Description = desc
		}
		descBuf = nil
	}

	// ensureUngroupedGroup makes sure the current column has at least one group so tasks appearing
	// before any H2 have somewhere to live. Always called at the start of a new column.
	ensureUngroupedGroup := func() {
		col := &board.Columns[curColumnIdx]
		if len(col.Groups) == 0 {
			col.Groups = append(col.Groups, model.Group{})
		}
		curGroupIdx = 0
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// H1 — new column. Takes precedence over everything.
		if strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "## ") {
			flushComment()
			flushDescription()
			sawFirstH1 = true
			title := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			board.Columns = append(board.Columns, model.Column{
				ID:    model.ColumnSlug(title),
				Title: title,
			})
			curColumnIdx = len(board.Columns) - 1
			ensureUngroupedGroup()
			curTaskIdx = -1
			continue
		}

		// H2 — new group within current column.
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			flushComment()
			flushDescription()
			if curColumnIdx < 0 {
				// Orphan H2 before any H1 — treat as preamble noise.
				preambleLines = append(preambleLines, line)
				continue
			}
			title := strings.TrimSpace(strings.TrimPrefix(line, "##"))
			col := &board.Columns[curColumnIdx]
			col.Groups = append(col.Groups, model.Group{Title: title})
			curGroupIdx = len(col.Groups) - 1
			curTaskIdx = -1
			continue
		}

		// H3 — new task.
		if strings.HasPrefix(line, "### ") {
			flushComment()
			flushDescription()
			if curColumnIdx < 0 {
				preambleLines = append(preambleLines, line)
				continue
			}
			rawTitle := strings.TrimSpace(strings.TrimPrefix(line, "###"))
			var id, branch string
			if m := taskIDPattern.FindStringSubmatchIndex(rawTitle); m != nil {
				id = rawTitle[m[2]:m[3]]
				// Strip the HTML comment from the visible title, collapsing surrounding whitespace.
				rawTitle = strings.TrimSpace(rawTitle[:m[0]] + rawTitle[m[1]:])
			}
			if m := taskBranchPattern.FindStringSubmatchIndex(rawTitle); m != nil {
				branch = rawTitle[m[2]:m[3]]
				rawTitle = strings.TrimSpace(rawTitle[:m[0]] + rawTitle[m[1]:])
			}
			task := model.Task{ID: id, Title: rawTitle, LinkedBranch: branch}
			col := &board.Columns[curColumnIdx]
			col.Groups[curGroupIdx].Tasks = append(col.Groups[curGroupIdx].Tasks, task)
			curTaskIdx = len(col.Groups[curGroupIdx].Tasks) - 1
			continue
		}

		// Before we see the first H1, everything goes into preamble.
		if !sawFirstH1 {
			preambleLines = append(preambleLines, line)
			continue
		}

		// At this point we're inside a column. If we don't have a task yet, anything else is
		// noise — drop it silently. (Blank lines between `# Col` and the first `###` are common.)
		if curTaskIdx < 0 {
			continue
		}

		// Blockquote lines accumulate into the current comment. A blank non-blockquote line
		// terminates any in-progress comment.
		if strings.HasPrefix(line, ">") {
			// If we had pending description text that hasn't been flushed, flush it before
			// transitioning into comment mode.
			if !inComment {
				flushDescription()
				inComment = true
			}
			commentBuf = append(commentBuf, line)
			continue
		}

		// Blank line: flush any comment in progress (ends one comment block), then treat as part
		// of the description if we're in description mode.
		if strings.TrimSpace(line) == "" {
			if inComment {
				flushComment()
			} else {
				descBuf = append(descBuf, line)
			}
			continue
		}

		// Any other line:
		// - If we're in a comment block, this starts new description prose. Flush the comment.
		//   The new prose gets appended to the task's description (option (c) from the design:
		//   preserve trailing content instead of losing it).
		// - Otherwise accumulate into description.
		if inComment {
			flushComment()
		}
		descBuf = append(descBuf, line)
	}

	// Final flush.
	flushComment()
	flushDescription()

	board.Preamble = strings.TrimRight(strings.Join(preambleLines, "\n"), "\n")
	// Strip the magic header line from the preamble if it's present — serializer always re-adds it.
	board.Preamble = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(board.Preamble), model.MagicHeader))

	return board, nil
}

// parseComment converts a contiguous blockquote block (lines starting with `>`) into a Comment.
// If the first line matches the canonical header pattern, timestamp and author are extracted;
// otherwise author is "" and timestamp is zero so hand-edited blockquotes round-trip as-is.
func parseComment(lines []string) model.Comment {
	// Strip the leading "> " (or ">" for empty lines) from each line.
	body := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimPrefix(l, ">")
		l = strings.TrimPrefix(l, " ")
		body = append(body, l)
	}
	joined := strings.Join(body, "\n")

	c := model.Comment{Body: joined}
	if len(body) == 0 {
		return c
	}
	if m := commentHeaderPattern.FindStringSubmatch(body[0]); m != nil {
		ts := strings.TrimSpace(m[1])
		author := strings.TrimSpace(m[2])
		rest := m[3]
		if t, err := time.Parse(model.CommentTimestampLayout, ts); err == nil {
			c.Timestamp = t
		}
		c.Author = author
		// Rebuild the body without the header prefix, preserving subsequent lines.
		if len(body) == 1 {
			c.Body = rest
		} else {
			c.Body = rest + "\n" + strings.Join(body[1:], "\n")
		}
		c.Body = strings.TrimRight(c.Body, "\n")
	}
	return c
}

// SerializeBoard renders a Board back to markdown. The output is deterministic: running
// ParseBoard → SerializeBoard → ParseBoard returns an equivalent Board.
func SerializeBoard(b model.Board) []byte {
	var buf bytes.Buffer
	buf.WriteString(model.MagicHeader)
	buf.WriteString("\n")
	if b.Preamble != "" {
		buf.WriteString("\n")
		buf.WriteString(b.Preamble)
		buf.WriteString("\n")
	}

	for ci, col := range b.Columns {
		if ci == 0 && b.Preamble == "" {
			buf.WriteString("\n")
		} else if ci > 0 {
			// Ensure a blank line before the next column.
			buf.WriteString("\n")
		}
		fmt.Fprintf(&buf, "# %s\n", col.Title)

		for gi, group := range col.Groups {
			if group.Title != "" {
				buf.WriteString("\n")
				fmt.Fprintf(&buf, "## %s\n", group.Title)
			}
			for _, task := range group.Tasks {
				buf.WriteString("\n")
				switch {
				case task.ID != "" && task.LinkedBranch != "":
					fmt.Fprintf(&buf, "### %s <!-- id: %s --> <!-- branch: %s -->\n", task.Title, task.ID, task.LinkedBranch)
				case task.ID != "":
					fmt.Fprintf(&buf, "### %s <!-- id: %s -->\n", task.Title, task.ID)
				case task.LinkedBranch != "":
					fmt.Fprintf(&buf, "### %s <!-- branch: %s -->\n", task.Title, task.LinkedBranch)
				default:
					fmt.Fprintf(&buf, "### %s\n", task.Title)
				}
				if strings.TrimSpace(task.Description) != "" {
					buf.WriteString("\n")
					buf.WriteString(strings.TrimRight(task.Description, "\n"))
					buf.WriteString("\n")
				}
				for _, c := range task.Comments {
					buf.WriteString("\n")
					writeComment(&buf, c)
				}
			}
			// Silence unused-index warning in case we later need it.
			_ = gi
		}
	}

	out := bytes.TrimRight(buf.Bytes(), "\n")
	out = append(out, '\n')
	return out
}

// writeComment renders a single comment as a blockquote block with every line prefixed by `> `.
// Empty body lines become `>` alone so the blockquote block stays contiguous.
func writeComment(buf *bytes.Buffer, c model.Comment) {
	// Build the first line: header + optional first body line.
	bodyLines := strings.Split(c.Body, "\n")
	first := bodyLines[0]
	rest := bodyLines[1:]

	if c.Author != "" && !c.Timestamp.IsZero() {
		fmt.Fprintf(buf, "> **[%s · %s]** %s\n",
			c.Timestamp.Format(model.CommentTimestampLayout),
			c.Author,
			first,
		)
	} else if c.Author != "" {
		// Author without a timestamp — still use the header shape so reparsing picks it up.
		fmt.Fprintf(buf, "> **[%s · %s]** %s\n",
			time.Time{}.Format(model.CommentTimestampLayout),
			c.Author,
			first,
		)
	} else {
		// No structured header — write the body as a raw blockquote so hand-authored comments
		// round-trip unchanged.
		if first == "" {
			buf.WriteString(">\n")
		} else {
			fmt.Fprintf(buf, "> %s\n", first)
		}
	}

	for _, line := range rest {
		if line == "" {
			buf.WriteString(">\n")
		} else {
			fmt.Fprintf(buf, "> %s\n", line)
		}
	}
}
