export interface ApiResponse<T = unknown> {
  data?: T;
  error?: {
    message: string;
    stack?: string;
  };
}
