interface SSEEvent {
  ID: string;
  Data: string;
  Event: string;
  Retry: string;
  Comment: string;
  Timestamp: number;
}

export default SSEEvent;