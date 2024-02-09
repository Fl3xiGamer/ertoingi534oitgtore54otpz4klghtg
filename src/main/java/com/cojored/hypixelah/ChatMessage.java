package com.cojored.hypixelah;

import net.minecraft.client.Minecraft;
import net.minecraft.network.chat.Component;
import net.minecraftforge.client.event.ClientChatEvent;
import net.minecraftforge.eventbus.api.SubscribeEvent;

public class ChatMessage {
    private final HypixelAH context;

    ChatMessage(HypixelAH context) {
        this.context = context;
    }

    @SubscribeEvent
    public void chatMessage(ClientChatEvent event) throws Exception {
        assert Minecraft.getInstance().player != null;
        if (event.getMessage().equals(".start")) {
            event.setCanceled(true);
            if (this.context.ws != null)
                this.context.ws.close();

            this.context.ws = new WebsocketClient(Minecraft.getInstance().player);
            this.context.ws.connect();
        } else if (event.getMessage().equals(".stop")) {
            event.setCanceled(true);
            this.context.ws.close();
        }
    }
}
