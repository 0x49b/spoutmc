package io.spoutmc.velocityplayers.model;

public final class PluginConfig {
    public String bindHost;
    public int port;
    public String token;
    /** Base URL for POST /api/v1/player/chat-ingest (e.g. http://127.0.0.1:3000/api/v1/player/chat-ingest) */
    public String spoutmcChatIngestUrl;
    /** Same value as SPOUT_PLAYER_CHAT_INGEST_SECRET on the SpoutMC API */
    public String spoutmcChatIngestSecret;
}
