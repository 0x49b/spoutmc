package io.spoutmc.velocityplayers.service;

import com.velocitypowered.api.proxy.Player;
import io.spoutmc.velocityplayers.model.ChatMessageRecord;
import io.spoutmc.velocityplayers.util.PluginUtils;
import net.kyori.adventure.text.Component;
import net.kyori.adventure.text.format.NamedTextColor;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

public final class ChatService {
    private static final Pattern PRIVATE_MESSAGE_COMMAND_PATTERN = Pattern.compile("^(msg|tell|w|whisper|m)\\s+(\\S+)\\s+(.+)$", Pattern.CASE_INSENSITIVE);

    private final Map<String, List<ChatMessageRecord>> chatHistoryByPlayer = new ConcurrentHashMap<>();
    private final Set<String> webReplyHandles = Set.of("admin", "mod", "moderator", "editor", "staff");

    public boolean handlePrivateMessageReplyAlias(Player player, String command) {
        if (command == null || command.isBlank()) {
            return false;
        }

        Matcher matcher = PRIVATE_MESSAGE_COMMAND_PATTERN.matcher(command.trim());
        if (!matcher.matches()) {
            return false;
        }

        String target = matcher.group(2);
        String message = matcher.group(3);
        if (target == null || message == null || message.isBlank()) {
            return false;
        }
        if (!isWebReplyTarget(target)) {
            return false;
        }

        if (captureIncomingReply(player, message)) {
            sendPlayerReplyFeedback(player, message.trim());
            return true;
        }
        return false;
    }

    public List<ChatMessageRecord> getChat(String playerName) {
        String normalized = PluginUtils.normalizeName(playerName);
        List<ChatMessageRecord> messages = chatHistoryByPlayer.getOrDefault(normalized, new ArrayList<>());
        synchronized (messages) {
            return new ArrayList<>(messages);
        }
    }

    public void sendStaffMessage(Player player, String sender, String role, String message) {
        String senderLabel = sender == null || sender.isBlank() ? "SpoutMC" : sender.trim();
        String roleLabel = role == null || role.isBlank() ? "staff" : role.trim();
        String normalized = PluginUtils.normalizeName(player.getUsername());
        List<ChatMessageRecord> chatMessages = chatHistoryByPlayer.computeIfAbsent(normalized, ignored -> new ArrayList<>());
        boolean firstStaffContact;
        synchronized (chatMessages) {
            firstStaffContact = chatMessages.stream().noneMatch(m -> "outgoing".equals(m.direction));
        }

        if (firstStaffContact) {
            player.sendMessage(Component.text(
                    "A staff member is contacting you privately through SpoutMC support chat.",
                    NamedTextColor.RED
            ));
            player.sendMessage(Component.text(
                    "You can reply with /r <message> or /reply <message>.",
                    NamedTextColor.RED
            ));
        }

        ChatMessageRecord entry = new ChatMessageRecord();
        entry.direction = "outgoing";
        entry.player = player.getUsername();
        entry.sender = senderLabel;
        entry.role = roleLabel;
        entry.message = message.trim();
        entry.timestamp = PluginUtils.nowIso();
        synchronized (chatMessages) {
            chatMessages.add(entry);
            trimChatHistory(chatMessages);
        }

        String formattedMessage = "[" + roleLabel.toUpperCase() + "] " + senderLabel + ": " + message;
        player.sendMessage(Component.text(formattedMessage, NamedTextColor.RED));
    }

    public boolean captureIncomingReply(Player player, String rawMessage) {
        String message = rawMessage == null ? "" : rawMessage.trim();
        if (message.isBlank()) {
            return false;
        }

        String normalizedPlayerName = PluginUtils.normalizeName(player.getUsername());
        List<ChatMessageRecord> chatMessages = chatHistoryByPlayer.get(normalizedPlayerName);
        if (chatMessages == null) {
            return false;
        }

        boolean hasOutgoingStaffMessage;
        synchronized (chatMessages) {
            hasOutgoingStaffMessage = chatMessages.stream().anyMatch(m -> "outgoing".equals(m.direction));
            if (!hasOutgoingStaffMessage) {
                return false;
            }

            ChatMessageRecord entry = new ChatMessageRecord();
            entry.direction = "incoming";
            entry.player = player.getUsername();
            entry.sender = player.getUsername();
            entry.role = "";
            entry.message = message;
            entry.timestamp = PluginUtils.nowIso();
            chatMessages.add(entry);
            trimChatHistory(chatMessages);
        }

        return true;
    }

    public void sendPlayerReplyFeedback(Player player, String message) {
        player.sendMessage(Component.text("Reply delivered to staff panel.", NamedTextColor.GRAY));
        player.sendMessage(Component.text("You -> Staff: " + message, NamedTextColor.GRAY));
    }

    private boolean isWebReplyTarget(String targetName) {
        if (targetName == null || targetName.isBlank()) {
            return false;
        }
        return webReplyHandles.contains(targetName.trim().toLowerCase());
    }

    private void trimChatHistory(List<ChatMessageRecord> chatMessages) {
        final int maxMessages = 200;
        if (chatMessages.size() <= maxMessages) {
            return;
        }
        chatMessages.subList(0, chatMessages.size() - maxMessages).clear();
    }
}
