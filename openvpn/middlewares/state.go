package middlewares

type State string

const STATE_UNDEFINED = State("UNDEFINED")
const STATE_CONNECTING = State("CONNECTING")
const STATE_WAIT = State("WAIT")
const STATE_AUTH = State("AUTH")
const STATE_GET_CONFIG = State("GET_CONFIG")
const STATE_ASSIGN_IP = State("ASSIGN_IP")
const STATE_ADD_ROUTES = State("ADD_ROUTES")
const STATE_CONNECTED = State("CONNECTED")
const STATE_RECONNECTING = State("RECONNECTING")
const STATE_EXITING = State("EXITING")
