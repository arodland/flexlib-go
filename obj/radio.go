package obj

import (
	"errors"
	"github.com/krippendorf/flexlib-go/sdrobjects"
	"github.com/krippendorf/flexlib-go/vita"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

type RadioData struct {
	Preampble *vita.VitaPacketPreamble
	Payload   []byte
	LastErr   error
}

type RadioContext struct {
	RadioAddr                string
	RadioCmdSeqNumber        int
	RadioConn                *net.TCPConn
	ChannelRadioData         chan *RadioData
	ChannelRadioResponse     chan string
	RadioHandle              string
	MyUdpEndpointIP          *net.IP
	MyUdpEndpointPort        string // we need strings for all cmds....
	ChannelVitaFFT           chan *sdrobjects.SdrFFTPacket
	ChannelVitaOpus          chan []byte
	ChannelVitaIfData        chan *sdrobjects.SdrIfData
	ChannelVitaMeter         chan *sdrobjects.SdrMeterPacket
	ChannelVitaWaterfallTile chan *sdrobjects.SdrWaterfallTile
	Panadapters              sync.Map
	IqStreams                sync.Map
	Debug                    bool
}

func getNextCommandPrefix(ctx *RadioContext) (string, int) {
	ctx.RadioCmdSeqNumber += 1
	return "C" + strconv.Itoa(ctx.RadioCmdSeqNumber) + "|", ctx.RadioCmdSeqNumber
}

func SendRadioCommand(ctx *RadioContext, cmd string) int {

	prefixString, sequence := getNextCommandPrefix(ctx)
	_, err := ctx.RadioConn.Write([]byte(prefixString + cmd + "\r"))

	if err != nil {
		panic(err)
	}

	return sequence
}

func GetOutboundIP() *net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return &localAddr.IP
}

func InitRadioContext(ctx *RadioContext) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", ctx.RadioAddr+":4992")

	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	// dial TCP connection to radio
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	ctx.RadioConn = conn

	ctx.MyUdpEndpointIP = GetOutboundIP()

	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	if err != nil {
		log.Println(err)
		panic(err)
	}

	go subscribeRadioUdp(ctx)
	go subscribeRadioUpdates(conn, ctx)

	// Subscribe data from radio
	SendRadioCommand(ctx, "sub tx all")
	SendRadioCommand(ctx, "sub atu all")
	SendRadioCommand(ctx, "sub amplifier all")
	SendRadioCommand(ctx, "sub meter all")
	SendRadioCommand(ctx, "sub pan all")
	SendRadioCommand(ctx, "sub slice all")
	SendRadioCommand(ctx, "sub gps all")
	SendRadioCommand(ctx, "sub audio_stream all")
	SendRadioCommand(ctx, "sub cwx all")
	SendRadioCommand(ctx, "sub xvtr all")
	SendRadioCommand(ctx, "sub memories all")
	SendRadioCommand(ctx, "sub daxiq all")
	SendRadioCommand(ctx, "sub dax all")
	SendRadioCommand(ctx, "sub usb_cable all")

	forever := make(chan bool)
	forever <- true

}

func subscribeRadioUpdates(conn *net.TCPConn, ctx *RadioContext) {

	l := log.New(os.Stderr, "RADIO_MSG", 0)
	buf := make([]byte, 4096)

	for {
		n, err := conn.Read(buf)

		if err != nil {
			continue
		}

		response := string(buf[:n])

		if len(response) == 0 {
			continue
		}

		lines := strings.Split(response, "\n")

		for _, responseLine := range lines {

			if len(strings.Trim(responseLine, " ")) == 0 {
				continue
			}

			if len(ctx.RadioHandle) == 0 && strings.HasPrefix(strings.ToUpper(responseLine), "H") {
				ctx.RadioHandle = responseLine[1:]
				l.Println("\nMY_RADIO_HANDLE>>" + ctx.RadioHandle)
			} else {

				if nil == ctx.ChannelRadioResponse {
					l.Println("Respnse Channel not bound: " + responseLine)
				} else {
					ctx.ChannelRadioResponse <- responseLine
					parseResponseLine(ctx, responseLine)
					if ctx.Debug {
						l.Println("DEBU:RESP:" + responseLine)
					}
				}
			}
		}

		if err != nil {
			l.Println(err)
		}
	}
}
func parseResponseLine(context *RadioContext, respLine string) {

	_, message := parseReplyStringPrefix(respLine)

	if strings.Contains(message, "display pan") {
		parsePanAdapterParams(context, message)
	} else if strings.Contains(message, "daxiq ") {
		parseDaxIqStatusParams(context, message)
	}
}

func parsePanAdapterParams(context *RadioContext, i string) {
	/*
		>0x40000000 wnb=0 wnb_level=50 wnb_updating=0 x_pixels=490 y_pixels=535 center=3.792057 bandwidth=0.885342 min_dbm=-126.84 max_dbm=-66.812 fps=5 average=70 weighted_average=0 rfgain=0 rxant=ANT2 wide=1 loopa=0 loopb=0 band=80 daxiq=0 daxiq_rate=0 capacity=16 available=16 waterfall=42000000 min_bw=0.004919999957085 max_b<>w=14.74560058594 xvtr= pre= ant_list=ANT1,ANT2,RX_A,XVTR<
	*/
	_, res, objectValue := parseKeyValueString(i, 1)

	var panadapter Panadapter
	panadapter.Id = objectValue
	dirty := false;

	actual, loaded := context.Panadapters.LoadOrStore(objectValue, panadapter)

	if(loaded){
		panadapter = actual.(Panadapter)
	}

	if val, ok := res["center"]; ok {
		rawFloatCenter, _ := strconv.ParseFloat(val, 64)
		panadapter.Center = int32(rawFloatCenter*1000000)
		dirty = true
	}

	if dirty {
		context.Panadapters.Store(objectValue, panadapter)
	}
}

func parseDaxIqStatusParams(context *RadioContext, i string) {


	_, res, objectValue := parseKeyValueString(i, 1)

	streamId, _ := strconv.Atoi(objectValue)
	var iqStream IqStream
	iqStream.Id = streamId
	dirty := false;

	actual, loaded := context.IqStreams.LoadOrStore(objectValue, iqStream)

	if(loaded){
		iqStream = actual.(IqStream)
	}

	if val, ok := res["pan"]; ok {
		iqStream.Pan = val
		dirty = true
	}

	if val, ok := res["rate"]; ok {
		iqStream.Rate, _ = strconv.Atoi(val)
		dirty = true
	}

	if dirty {
		context.IqStreams.Store(objectValue, iqStream)
	}
}

func parseReplyStringPrefix(in string) (string, string) {
	var prefix string
	var message string

	tokens := strings.Split(in, "|")

	if len(tokens) == 2 {
		return tokens[0], tokens[1]
	}

	return prefix, message
}

func parseKeyValueString(in string, words int) (error, map[string]string, string) {

	var res map[string]string
	res = map[string]string{}

	tokens := strings.Split(in, " ")

	if len(tokens) == 0 {
		return errors.New("no tokens found"), res, ""
	}

	if strings.Index(in, "=") < 0 {
		return errors.New("not a key value list"), res, ""
	}

	skipedWords := 0
	var objectValue string
	for rngAttr := range tokens[:] {

		contentTokens := strings.Split(tokens[rngAttr], " ")

		for cntToken := range contentTokens {

			keyValueTokens := strings.Split(contentTokens[cntToken], "=")

			if len(keyValueTokens) == 2 {
				res[keyValueTokens[0]] = keyValueTokens[1]
			} else if len(keyValueTokens) == 1 {

				if skipedWords == words { // first prefix is object identifier itself
					objectValue = keyValueTokens[0]
				} else {
					skipedWords++
				}
			}
		}
	}

	return nil, res, objectValue
}

func subscribeRadioUdp(ctx *RadioContext) {

	FLexBroadcastAddr, err := net.ResolveUDPAddr("udp", ctx.MyUdpEndpointIP.String()+":"+ctx.MyUdpEndpointPort)

	if err != nil {
		panic(err)
	}

	ServerConn, err := net.ListenUDP("udp", FLexBroadcastAddr)

	if err != nil {
		panic(err)
	}

	defer ServerConn.Close()
	buf := make([]byte, 64000)

	if err != nil {
		panic(err)
	}

	if err != nil {
		panic(err)
	}

	for {
		n, _, _ := ServerConn.ReadFromUDP(buf)
		radioData := new(RadioData)
		radioData.LastErr, radioData.Preampble, radioData.Payload = vita.ParseVitaPreamble(buf[:n])
		if ctx.ChannelRadioData != nil {
			ctx.ChannelRadioData <- radioData
		}

		dispatchDataToChannels(ctx, radioData)
	}
}

func dispatchDataToChannels(ctx *RadioContext, data *RadioData) {
	switch data.Preampble.Header.Pkt_type {

	case vita.ExtDataWithStream:

		switch data.Preampble.Class_id.PacketClassCode {

		case vita.SL_VITA_FFT_CLASS:
			if nil != ctx.ChannelVitaFFT {
				ctx.ChannelVitaFFT <- vita.ParseVitaFFT(data.Payload, data.Preampble)
			}
			break
		case vita.SL_VITA_OPUS_CLASS:
			if nil != ctx.ChannelVitaOpus {
				ctx.ChannelVitaOpus <- data.Payload[:len(data.Payload)-data.Preampble.Header.Payload_cutoff_bytes]
			}
			break
		case vita.SL_VITA_IF_NARROW_CLASS:
			if nil != ctx.ChannelVitaIfData {
				vita.ParseFData(data.Payload, data.Preampble)
			}
			break
		case vita.SL_VITA_METER_CLASS:
			if nil != ctx.ChannelVitaMeter {
				ctx.ChannelVitaMeter <- vita.ParseVitaMeterPacket(data.Payload, data.Preampble)
			}
			break
		case vita.SL_VITA_DISCOVERY_CLASS:
			// maybe later - we use static addresses
			break
		case vita.SL_VITA_WATERFALL_CLASS:
			if nil != ctx.ChannelVitaWaterfallTile {
				vita.ParseVitaWaterfall(data.Payload, data.Preampble)
			}
			break
		default:
			break
		}

		break

	case vita.IFDataWithStream:
		switch data.Preampble.Class_id.PacketClassCode {
		case vita.SL_VITA_IF_WIDE_CLASS_24kHz:
			fallthrough
		case vita.SL_VITA_IF_WIDE_CLASS_48kHz:
			fallthrough
		case vita.SL_VITA_IF_WIDE_CLASS_96kHz:
			fallthrough
		case vita.SL_VITA_IF_WIDE_CLASS_192kHz:
			if nil != ctx.ChannelVitaIfData {
				ctx.ChannelVitaIfData <- vita.ParseFData(data.Payload, data.Preampble)
			}
		}
		break
	}
}