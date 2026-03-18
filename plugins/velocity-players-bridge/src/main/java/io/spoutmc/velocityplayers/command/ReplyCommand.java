package io.spoutmc.velocityplayers.command;

import com.velocitypowered.api.command.SimpleCommand;
import com.velocitypowered.api.proxy.Player;
import io.spoutmc.velocityplayers.service.ChatService;
import net.kyori.adventure.text.Component;

public final class ReplyCommand implements SimpleCommand {
    private final ChatService chatService;

    public ReplyCommand(ChatService chatService) {
        this.chatService = chatService;
    }

    @Override
    public void execute(Invocation invocation) {
        if (!(invocation.source() instanceof Player player)) {
            invocation.source().sendMessage(Component.text("This command can only be used by players."));
            return;
        }

        String message = String.join(" ", invocation.arguments()).trim();
        if (message.isBlank()) {
            player.sendMessage(Component.text("Usage: /reply <message>"));
            return;
        }

        if (chatService.captureIncomingReply(player, message)) {
            chatService.sendPlayerReplyFeedback(player, message);
            return;
        }

        player.sendMessage(Component.text("No active staff chat. Use /msg admin <message> first."));
    }
}
