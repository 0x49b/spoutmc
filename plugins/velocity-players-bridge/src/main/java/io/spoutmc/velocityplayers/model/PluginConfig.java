package io.spoutmc.velocityplayers.model;

public final class PluginConfig {
    public String bindHost;
    public int port;
    public String token;
    /**
     * Base URL for POST /api/v1/player/chat-ingest.
     * Use the address reachable from this JVM: on the same host as SpoutMC, {@code http://127.0.0.1:3000/...} works.
     * If Velocity runs in Docker and SpoutMC on the host, use {@code http://host.docker.internal:3000/...}
     * (Docker Desktop; on Linux add {@code extra_hosts: host.docker.internal:host-gateway}).
     */
    public String spoutmcChatIngestUrl;
    /** Same value as SPOUT_PLAYER_CHAT_INGEST_SECRET on the SpoutMC API */
    public String spoutmcChatIngestSecret;
}
