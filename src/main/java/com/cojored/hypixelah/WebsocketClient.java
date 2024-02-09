package com.cojored.hypixelah;

import net.minecraft.ChatFormatting;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.components.ChatComponent;
import net.minecraft.client.player.LocalPlayer;
import net.minecraft.network.chat.*;
import net.minecraft.network.chat.Component;
import net.minecraft.server.MinecraftServer;
import net.minecraftforge.client.ClientCommandHandler;
import net.minecraftforge.client.event.ClientChatEvent;
import org.java_websocket.client.WebSocketClient;
import org.java_websocket.handshake.ServerHandshake;
import org.json.JSONObject;

import java.awt.*;
import java.net.URI;
import java.net.URISyntaxException;
import java.util.Objects;

public class WebsocketClient {
    private static final char[] c = new char[]{'K', 'M', 'B', 'T'};
    private final LocalPlayer player;
    private WebSocketClient client;

    WebsocketClient(LocalPlayer player) {
        this.player = player;
    }

    private static String coolFormat(double n, int iteration) {
        double d = ((long) n / 100) / 10.0;
        boolean isRound = (d * 10) % 10 == 0;
        return d < 1000 ?
                (isRound || d > 9.99 ?
                        (int) d * 10 / 10 : d + ""
                ) + "" + c[iteration]
                : coolFormat(d, iteration + 1);

    }

    public void connect() {
        URI uri;
        try {
            uri = new URI("wss://vjsdz2-8080.csb.app/socket");
        } catch (URISyntaxException e) {
            e.printStackTrace();
            return;
        }

        this.client = new WebSocketClient(uri) {
            @Override
            public void onOpen(ServerHandshake serverHandshake) {
                player.sendSystemMessage(Component.literal("AH Sniper Started!").withStyle(style -> style.withColor(ChatFormatting.GREEN)));
            }

            @Override
            public void onMessage(String s) {
                String message = s;
                Boolean gui = false;

                if (!Objects.equals(message, "ping")) {
                    if(message.startsWith("gui")) {
                        message = message.split("gui")[1];
                        gui=true;
                    }
                    JSONObject obj = new JSONObject(message);

                    String price = coolFormat(obj.getDouble("price"), 0);
                    String oldLow = coolFormat(obj.getDouble("low"), 0);
                    String profit = coolFormat(obj.getDouble("profit"), 0);
                    String profit = coolFormat(obj.getDouble("profitAfterTax"), 0);
                    String name = obj.getString("name");
                    String uuid = obj.getString("uuid");

                    System.out.println(gui);
                    if(gui)
                        player.connection.sendChat("/viewauction " + uuid);

                    ClickEvent cE1 = new ClickEvent(ClickEvent.Action.RUN_COMMAND, "/viewauction " + uuid);
                    HoverEvent hE1 = new HoverEvent(HoverEvent.Action.SHOW_TEXT, Component.nullToEmpty("/viewauction " + uuid));
                    Component msg = Component.empty()
                            .append(Component.literal(name).withStyle(style -> style.withColor(ChatFormatting.DARK_PURPLE)))
                            .append(" | ")
                            .append(Component.literal(price + " -> " + oldLow).withStyle(style -> style.withColor(ChatFormatting.AQUA)))
                            .append(" | ")
                            .append(Component.literal("+" + profit).withStyle(style -> style.withColor(ChatFormatting.GREEN)))
                            .append(Component.literal(" [BUY NOW]").withStyle(style -> style.withColor(ChatFormatting.GOLD).withClickEvent(cE1).withHoverEvent(hE1)));
                    player.sendSystemMessage(msg);
                }
            }

            @Override
            public void onClose(int i, String s, boolean b) {
                player.sendSystemMessage(Component.literal("AH Sniper Connection Lost.").withStyle(style -> style.withColor(ChatFormatting.RED)));
            }

            @Override
            public void onError(Exception e) {
            }
        };
        this.client.setConnectionLostTimeout(0);
        this.client.connect();
    }

    public void close() {
        this.client.close();
    }
}
