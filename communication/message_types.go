package communication

type MessageType string

// New Node was registered in network
const NODE_REGISTER = MessageType("node-register")

// Client is trying to establish new dialog with Node
const DIALOG_CREATE = MessageType("dialog-create")
