from neonize.client import NewClient
from neonize.utils import build_jid
from neonize.events import MessageEv, ConnectedEv, event

client = NewClient("bot")

@client.event
def on_connected(client: NewClient, event: ConnectedEv):
    print("Bot connected successfully!")
    
@client.event  
def on_message(client: NewClient, event: MessageEv):
    return
    if event.message.conversation == "hi":
        client.reply_message("Hello! 👋", event.message)


jid = build_jid("+4917691334407")
client.connect()
client.send_message(jid, "test")