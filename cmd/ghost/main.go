package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
)

func main() {
	region := flag.String("region", "us-east-1", "AWS region to scan")
	flag.Parse()

	ctx := context.Background()
	log.Printf("Connecting to AWS (Region: %s)...", *region)

	clients, err := awsclient.New(ctx, *region)
	if err != nil {
		log.Fatalf("Client initialization error: %v", err)
	}

	fmt.Printf("\n[1/2] Scanning unattached EBS volumes...\n")
	ebsVolumes, err := collector.CollectUnattachedEBS(ctx, clients.EC2, *region)
	if err != nil {
		log.Fatalf("EBS collection failed: %v", err)
	}
	fmt.Printf("Found %d unattached volume(s)\n", len(ebsVolumes))

	fmt.Printf("\n[2/2] Scanning running RDS instances...\n")
	rdsInstances, err := collector.CollectRunningRDS(ctx, clients.RDS, *region)
	if err != nil {
		log.Fatalf("RDS collection failed: %v", err)
	}
	fmt.Printf("Found %d running RDS instance(s)\n", len(rdsInstances))

	allResources := append(ebsVolumes, rdsInstances...)

	// Print summary table to stdout
	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "RESOURCE ID\tTYPE\tSPEC\tSTATUS\tEST. MONTHLY WASTE")
	fmt.Fprintln(w, "-----------\t----\t----\t------\t------------------")

	var totalWaste float64
	for _, r := range allResources {
		spec := fmt.Sprintf("%d GB (%s)", r.SizeGB, r.StorageType)
		if r.Type == collector.TypeRDSInstance {
			spec = fmt.Sprintf("%s / %d GB", r.InstanceType, r.SizeGB)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t$%.2f\n", r.ID, r.Type, spec, r.Status, r.EstimatedMo)
		totalWaste += r.EstimatedMo
	}
	w.Flush()

	fmt.Printf("\nTotal Potential Monthly Spend Detected: $%.2f\n\n", totalWaste)
}
