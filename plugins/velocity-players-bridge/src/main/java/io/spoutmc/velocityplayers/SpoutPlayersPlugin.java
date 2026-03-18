package io.spoutmc.velocityplayers;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.inject.Inject;
import com.sun.net.httpserver.HttpServer;
import com.velocitypowered.api.command.CommandMeta;
import com.velocitypowered.api.event.ResultedEvent;
import com.velocitypowered.api.event.Subscribe;
import com.velocitypowered.api.event.command.CommandExecuteEvent;
import com.velocitypowered.api.event.connection.DisconnectEvent;
import com.velocitypowered.api.event.connection.LoginEvent;
import com.velocitypowered.api.event.connection.PostLoginEvent;
import com.velocitypowered.api.event.player.ServerConnectedEvent;
import com.velocitypowered.api.event.proxy.ProxyInitializeEvent;
import com.velocitypowered.api.event.proxy.ProxyShutdownEvent;
import com.velocitypowered.api.plugin.Plugin;
import com.velocitypowered.api.plugin.annotation.DataDirectory;
import com.velocitypowered.api.proxy.Player;
import com.velocitypowered.api.proxy.ProxyServer;
import io.spoutmc.velocityplayers.api.BridgeApiHandler;
import io.spoutmc.velocityplayers.command.ReplyCommand;
import io.spoutmc.velocityplayers.model.BanRecord;
import io.spoutmc.velocityplayers.model.PluginConfig;
import io.spoutmc.velocityplayers.service.ChatService;
import io.spoutmc.velocityplayers.service.ConfigService;
import io.spoutmc.velocityplayers.service.PlayerStateService;
import net.kyori.adventure.text.Component;
import org.slf4j.Logger;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.concurrent.Executors;

@Plugin(
        id = "spoutmc-players",
        name = "SpoutMC Players Bridge",
        version = "0.1.0",
        description = "Tracks players and exposes HTTP/SSE API for SpoutMC",
        authors = {"SpoutMC"}
)
public final class SpoutPlayersPlugin {
    private final ProxyServer proxy;
    private final Logger logger;
    private final Path dataDirectory;
    private final Gson gson;
    private final ChatService chatService;
    private final ConfigService configService;
    private final PlayerStateService playerStateService;

    private HttpServer httpServer;
    private PluginConfig config;
    private BridgeApiHandler apiHandler;
    private CommandMeta replyCommandMeta;

    @Inject
    public SpoutPlayersPlugin(ProxyServer proxy, Logger logger, @DataDirectory Path dataDirectory) {
        this.proxy = proxy;
        this.logger = logger;
        this.dataDirectory = dataDirectory;
        this.gson = new GsonBuilder().setPrettyPrinting().create();
        this.chatService = new ChatService();
        this.configService = new ConfigService(dataDirectory);
        this.playerStateService = new PlayerStateService(proxy, logger, gson, dataDirectory);
    }

    @Subscribe
    public void onInitialize(ProxyInitializeEvent event) {
        try {
            Files.createDirectories(dataDirectory);
            this.config = configService.loadConfig();
            playerStateService.loadState();
            playerStateService.seedOnlinePlayers();
            this.apiHandler = new BridgeApiHandler(logger, gson, config, playerStateService, chatService);
            startHttpServer();
            registerCommands();
            apiHandler.broadcastSnapshot();
            logger.info("[SpoutPlayers] Plugin initialized on {}:{}", config.bindHost, config.port);
        } catch (Exception e) {
            logger.error("[SpoutPlayers] Failed to initialize plugin", e);
        }
    }

    @Subscribe
    public void onShutdown(ProxyShutdownEvent event) {
        unregisterCommands();
        stopHttpServer();
        playerStateService.saveState();
        logger.info("[SpoutPlayers] Plugin shutdown complete");
    }

    @Subscribe
    public void onLogin(LoginEvent event) {
        BanRecord ban = playerStateService.getBan(event.getPlayer().getUsername());
        if (ban != null) {
            String reason = ban.reason == null || ban.reason.isBlank() ? "Banned from this network" : ban.reason;
            event.setResult(ResultedEvent.ComponentResult.denied(Component.text("You are banned: " + reason)));
        }
    }

    @Subscribe
    public void onPostLogin(PostLoginEvent event) {
        playerStateService.onPostLogin(event.getPlayer());
        playerStateService.saveState();
        if (apiHandler != null) {
            apiHandler.broadcastSnapshot();
        }
    }

    @Subscribe
    public void onServerConnected(ServerConnectedEvent event) {
        playerStateService.onServerConnected(event.getPlayer(), event.getServer().getServerInfo().getName());
        playerStateService.saveState();
        if (apiHandler != null) {
            apiHandler.broadcastSnapshot();
        }
    }

    @Subscribe
    public void onDisconnect(DisconnectEvent event) {
        playerStateService.onDisconnect(event.getPlayer());
        playerStateService.saveState();
        if (apiHandler != null) {
            apiHandler.broadcastSnapshot();
        }
    }

    @Subscribe
    public void onCommandExecute(CommandExecuteEvent event) {
        if (!(event.getCommandSource() instanceof Player player)) {
            return;
        }

        if (chatService.handlePrivateMessageReplyAlias(player, event.getCommand())) {
            event.setResult(CommandExecuteEvent.CommandResult.denied());
        }
    }

    private void startHttpServer() throws IOException {
        httpServer = HttpServer.create(new InetSocketAddress(config.bindHost, config.port), 0);
        httpServer.createContext("/", apiHandler);
        httpServer.setExecutor(Executors.newCachedThreadPool(runnable -> {
            Thread t = new Thread(runnable, "spoutmc-players-http");
            t.setUncaughtExceptionHandler((thread, throwable) ->
                    logger.error("[SpoutPlayers] Uncaught exception in HTTP thread {}", thread.getName(), throwable));
            return t;
        }));
        httpServer.start();
    }

    private void stopHttpServer() {
        if (apiHandler != null) {
            apiHandler.closeAllSseClients();
        }
        if (httpServer != null) {
            httpServer.stop(0);
        }
    }

    private void registerCommands() {
        try {
            replyCommandMeta = proxy.getCommandManager()
                    .metaBuilder("reply")
                    .aliases("r")
                    .plugin(this)
                    .build();
            proxy.getCommandManager().register(replyCommandMeta, new ReplyCommand(chatService));
        } catch (Exception e) {
            logger.error("[SpoutPlayers] Failed to register /reply command", e);
        }
    }

    private void unregisterCommands() {
        if (replyCommandMeta == null) {
            return;
        }
        try {
            proxy.getCommandManager().unregister(replyCommandMeta);
        } catch (Exception e) {
            logger.warn("[SpoutPlayers] Failed to unregister /reply command", e);
        } finally {
            replyCommandMeta = null;
        }
    }
}
