package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/google/trillian"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	adminServer = flag.String("admin_server", "", "Address of the Trillian admin server (e.g., proofpix-trillian-log-server-abc-uc.a.run.app:443)")
	kmsKeyURI   = flag.String("kms_key_uri", "", "Full resource name of the Cloud KMS signing key (e.g., gcp-kms://projects/.../cryptoKeys/...)")
)

func main() {
	flag.Parse()

	// Validate required flags
	if *adminServer == "" {
		log.Fatal("--admin_server flag is required")
	}
	if *kmsKeyURI == "" {
		log.Fatal("--kms_key_uri flag is required")
	}

	log.Println("ProofPix Trillian Tree Provisioning Tool")
	log.Printf("Admin Server: %s", *adminServer)
	log.Printf("KMS Key URI: %s", *kmsKeyURI)

	// Create secure gRPC connection
	log.Println("Creating secure gRPC connection...")
	creds := credentials.NewTLS(&tls.Config{
		ServerName: *adminServer,
	})

	conn, err := grpc.Dial(*adminServer, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("Failed to connect to admin server: %v", err)
	}
	defer conn.Close()

	// Create Trillian admin client
	log.Println("Creating Trillian admin client...")
	adminClient := trillian.NewTrillianAdminClient(conn)

	// Construct the tree object
	log.Println("Constructing tree configuration...")
	tree := &trillian.Tree{
		TreeType:    trillian.TreeType_LOG,
		TreeState:   trillian.TreeState_ACTIVE,
		DisplayName: "ProofPix Authenticity Log",
		Description: fmt.Sprintf("ProofPix authenticity log using KMS key: %s", *kmsKeyURI),
	}

	// Create the tree creation request
	log.Println("Creating tree creation request...")
	request := &trillian.CreateTreeRequest{
		Tree: tree,
	}

	// Call CreateTree with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Sending create tree request...")
	response, err := adminClient.CreateTree(ctx, request)
	if err != nil {
		log.Fatalf("Failed to create tree: %v", err)
	}

	// Print the tree ID on success
	log.Printf("Tree created successfully!")
	fmt.Printf("Tree ID: %d\n", response.TreeId)
	log.Printf("Tree Display Name: %s", response.DisplayName)
	log.Printf("Tree State: %s", response.TreeState.String())
	log.Printf("KMS Key URI (for signer configuration): %s", *kmsKeyURI)
}