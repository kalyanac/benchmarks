package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"text/tabwriter"
	"time"

	pp "github.com/kalyanac/benchmarks/pingpong/proto"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var (
	startMode      = flag.String("mode", "server", "start process as server or client")
	serverAddress  = flag.String("server_address", "localhost:8080", "server address as hostname:port")
	warmUp         = flag.Bool("warmup", true, "run warm up before benchmarking")
	numIterations  = flag.Int("iterations", 100000, "number of iterations to run for each message size")
	messageSize    = flag.Int64("messae_size", -1, "message size in bytes. -1 to run all message sizes.")
	maxMessageSize = flag.Int64("max_message_size", 1<<22, "maximum message size limit")

	warmIterations = 50
)

type server struct {
	pp.UnimplementedPingPongServer
}

func startServer() {
	lis, err := net.Listen("tcp", *serverAddress)
	if err != nil {
		log.Fatalf(err.Error())
	}
	s := grpc.NewServer()
	pp.RegisterPingPongServer(s, &server{})

	s.Serve(lis)
}

func (s *server) EmptyPingPong(ctx context.Context, in *empty.Empty) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (s *server) PingPong(ctx context.Context, in *pp.PingPongMessage) (*pp.PingPongMessage, error) {
	return &pp.PingPongMessage{Payload: in.Payload}, nil

}

func startClient() {
	conn, err := grpc.Dial(*serverAddress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer conn.Close()
	c := pp.NewPingPongClient(conn)
	ctx := context.Background()

	if *warmUp {
		log.Println("Warm up mode enabled.")
	}
	var payload []byte
	var size int64
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 1, '\t', 0)
	fmt.Fprintf(w, "\n %s\t%s", "bytes", "latency")

	for size = 0; size <= *maxMessageSize; size *= 2 {
		payload = make([]byte, size)
		rand.Read(payload)
		if *warmUp {
			//			fmt.Printf("Starting warm up for %v bytes. \n", size)
			for i := 0; i < warmIterations; i++ {
				_, err := c.PingPong(ctx, &pp.PingPongMessage{Payload: payload})
				if err != nil {
					log.Fatalf(err.Error())
				}
			}
			//			fmt.Printf("Finished warm up for %v bytes.\n", size)
		}
		startTime := time.Now()
		for i := 0; i < *numIterations; i++ {
			_, err := c.PingPong(ctx, &pp.PingPongMessage{Payload: payload})
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		fmt.Fprintf(w, "\n %v\t%v", size, time.Now().Sub(startTime)/(time.Duration(*numIterations)*time.Nanosecond))
		w.Flush()
		//		fmt.Println("Message size: ", size, " duration: ", time.Now().Sub(startTime)/(time.Duration(*numIterations)*time.Nanosecond))
		if size == 0 {
			size = 1
		}
	}
}

func main() {
	flag.Parse()
	if *startMode == "server" {
		log.Println("Start process in server mode")
		startServer()
	} else if *startMode == "client" {
		startClient()
	} else {
		log.Fatalf("mode %s is not valid", *startMode)
	}
}
