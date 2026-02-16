import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import Home from '../app/page';

describe('Home', () => {
  it('renders the welcome heading', () => {
    render(<Home />);

    expect(screen.getByText('Welcome')).toBeDefined();
  });
});
