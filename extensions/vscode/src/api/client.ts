import axios, { AxiosInstance } from "axios";
import WebSocket from "ws";

export interface Message {
  role: "user" | "assistant" | "system";
  content: string;
}

export interface ChatRequest {
  messages: Message[];
  model?: string;
  stream?: boolean;
  id?: string;
}

export interface ChatResponse {
  id: string;
  message: Message;
  created: string;
}

export interface StreamChunk {
  id: string;
  content: string;
  done: boolean;
}

export interface ToolRequest {
  name: string;
  arguments: Record<string, any>;
}

export interface ToolResponse {
  id: string;
  result?: string;
  error?: string;
}

export interface Conversation {
  id: string;
  title: string;
  created: string;
  updated: string;
  messages?: Message[];
}

export interface HistoryResponse {
  conversations: Conversation[];
  total: number;
  page: number;
  per_page: number;
}

export class RubrDuckAPI {
  private client: AxiosInstance;
  private websocket?: WebSocket;

  constructor(private serverUrl: string, private authToken: string) {
    this.client = axios.create({
      baseURL: serverUrl,
      timeout: 30000,
      headers: {
        "Content-Type": "application/json",
        ...(authToken && { Authorization: `Bearer ${authToken}` }),
      },
    });
  }

  updateConfig(serverUrl: string, authToken: string) {
    this.serverUrl = serverUrl;
    this.authToken = authToken;

    this.client = axios.create({
      baseURL: serverUrl,
      timeout: 30000,
      headers: {
        "Content-Type": "application/json",
        ...(authToken && { Authorization: `Bearer ${authToken}` }),
      },
    });

    // Close existing websocket
    if (this.websocket) {
      this.websocket.close();
      this.websocket = undefined;
    }
  }

  async healthCheck(): Promise<boolean> {
    try {
      const response = await this.client.get("/health");
      return response.status === 200;
    } catch (error) {
      return false;
    }
  }

  async sendChat(request: ChatRequest): Promise<ChatResponse> {
    const response = await this.client.post<ChatResponse>("/chat", request);
    return response.data;
  }

  async *streamChat(request: ChatRequest): AsyncIterable<StreamChunk> {
    const response = await this.client.post(
      "/stream",
      { ...request, stream: true },
      {
        responseType: "stream",
      }
    );

    let buffer = "";

    for await (const chunk of response.data) {
      buffer += chunk.toString();
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (line.startsWith("data: ")) {
          const data = line.slice(6);
          if (data === "[DONE]") {
            return;
          }
          try {
            const streamChunk: StreamChunk = JSON.parse(data);
            yield streamChunk;
          } catch (error) {
            console.error("Failed to parse stream chunk:", error);
          }
        }
      }
    }
  }

  async executeTool(request: ToolRequest): Promise<ToolResponse> {
    const response = await this.client.post<ToolResponse>("/tools", request);
    return response.data;
  }

  async getHistory(page = 1, perPage = 20): Promise<HistoryResponse> {
    const response = await this.client.get<HistoryResponse>("/history", {
      params: { page, per_page: perPage },
    });
    return response.data;
  }

  async getConversation(id: string): Promise<Conversation | null> {
    try {
      const response = await this.client.get<HistoryResponse>("/history", {
        params: { id },
      });
      return response.data.conversations[0] || null;
    } catch (error) {
      return null;
    }
  }

  connectWebSocket(
    onMessage: (chunk: StreamChunk) => void,
    onError?: (error: Error) => void
  ): void {
    if (this.websocket) {
      this.websocket.close();
    }

    const wsUrl = this.serverUrl.replace(/^http/, "ws") + "/ws";
    this.websocket = new WebSocket(wsUrl, {
      headers: this.authToken
        ? { Authorization: `Bearer ${this.authToken}` }
        : undefined,
    });

    const ws = this.websocket;

    ws.on("message", (data: WebSocket.RawData) => {
      try {
        const chunk: StreamChunk = JSON.parse(data.toString());
        onMessage(chunk);
      } catch (error) {
        console.error("Failed to parse WebSocket message:", error);
      }
    });

    ws.on("error", (error: Error) => {
      console.error("WebSocket error:", error);
      if (onError) {
        onError(error);
      }
    });

    ws.on("close", () => {
      console.log("WebSocket connection closed");
    });
  }

  sendWebSocketMessage(message: any): void {
    if (this.websocket && this.websocket.readyState === WebSocket.OPEN) {
      this.websocket.send(JSON.stringify(message));
    }
  }

  dispose(): void {
    if (this.websocket) {
      this.websocket.close();
    }
  }
}
