package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// 1. Define your Cloud Run address.
	// IMPORTANT: Strip the "https://" prefix and specify port 443.
	serverAddress := "my-grpc-service-xyz.run.app:443"

	// 2. Fetch the host system's native root certificate pool.
	// Google Cloud Run uses standard public certificates, so your local machine
	// already trusts the Certificate Authority (CA) out of the box.
	certPool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("Failed to load system root CA certificates: %v", err)
	}

	// 3. Create secure TLS credentials targeting your Cloud Run domain name.
	creds := credentials.NewTLS(&tls.Config{
		RootCAs: certPool,
	})

	// 4. Establish a secure connection to the Cloud Run load balancer.
	log.Printf("Connecting securely to %s...", serverAddress)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, serverAddress, 
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(), // Blocks until the connection is successfully established
	)
	if err != nil {
		log.Fatalf("Could not connect to gRPC server: %v", err)
	}
	defer conn.Close()
	log.Println("Successfully connected to Cloud Run gRPC server!")

	// 5. Initialize your generated client stub and invoke your RPCs here.
	// exampleClient := pb.NewYourServiceClient(conn)
	// response, err := exampleClient.YourMethod(context.Background(), &pb.YourRequest{})
}