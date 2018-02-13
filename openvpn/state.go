package openvpn

type State string

const UndefinedState = State("UNDEFINED")
const ConnectingState = State("CONNECTING")
const WaitState = State("WAIT")
const AuthenticatingState = State("AUTH")
const GetConfigState = State("GET_CONFIG")
const AssignIpState = State("ASSIGN_IP")
const AddRoutesState = State("ADD_ROUTES")
const ConnectedState = State("CONNECTED")
const ReconnectingState = State("RECONNECTING")
const ExitingState = State("EXITING")
