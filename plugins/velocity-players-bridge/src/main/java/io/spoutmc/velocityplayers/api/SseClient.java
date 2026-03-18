package io.spoutmc.velocityplayers.api;

import com.sun.net.httpserver.HttpExchange;

import java.io.IOException;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;

final class SseClient {
    private final HttpExchange exchange;
    private final OutputStream outputStream;
    private final Object lock = new Object();
    private volatile boolean closed;

    SseClient(HttpExchange exchange) throws IOException {
        this.exchange = exchange;
        this.outputStream = exchange.getResponseBody();
        this.closed = false;
    }

    boolean sendSse(String jsonPayload) {
        String line = "data: " + jsonPayload + "\n\n";
        return writeLine(line);
    }

    boolean sendComment(String comment) {
        return writeLine(": " + comment + "\n\n");
    }

    boolean isClosed() {
        return closed;
    }

    void close() {
        synchronized (lock) {
            if (closed) {
                return;
            }
            closed = true;
            try {
                outputStream.close();
            } catch (IOException ignored) {
            }
            exchange.close();
        }
    }

    private boolean writeLine(String payload) {
        synchronized (lock) {
            if (closed) {
                return false;
            }
            try {
                outputStream.write(payload.getBytes(StandardCharsets.UTF_8));
                outputStream.flush();
                return true;
            } catch (IOException ignored) {
                close();
                return false;
            }
        }
    }
}
