package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/abhiraj-ku/finops/internal/attribution"
	"github.com/abhiraj-ku/finops/internal/awsclient"
	"github.com/abhiraj-ku/finops/internal/collector"
	"github.com/abhiraj-ku/finops/internal/remediation"
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

	auditor := collector.NewTelemetryAuditor(clients.CloudWatch)

	fmt.Printf("\n[step 1/2] Auditing EBS volumes...\n")
	ebsCandidates, err := collector.UnattachedVolumes(ctx, clients.EC2, *region)
	if err != nil {
		log.Fatalf("EBS collection failed: %v", err)
	}
	verifiedEBS, err := auditor.FilterIdleEBS(ctx, ebsCandidates)
	if err != nil {
		log.Fatalf("EBS telemetry audit failed: %v", err)
	}
	fmt.Printf("-> Found %d unattached volume(s) with 0 IOPS over 14 days\n", len(verifiedEBS))

	fmt.Printf("\n[step 2/2] Auditing RDS instances...\n")
	rdsCandidates, err := collector.RunningRdsInfo(ctx, clients.RDS, *region)
	if err != nil {
		log.Fatalf("RDS collection failed: %v", err)
	}
	verifiedRDS, err := auditor.FilterIdleRDS(ctx, rdsCandidates)
	if err != nil {
		log.Fatalf("RDS telemetry audit failed: %v", err)
	}
	fmt.Printf("-> Found %d RDS instance(s) with 0 connections over 14 days\n", len(verifiedRDS))

	allResources := append(verifiedEBS, verifiedRDS...)

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

	tracker := attribution.NewTracker(clients.CloudTrail)
	fmt.Println("\n[3/3] fetching owner from cloudtrail..\n")

	for i, r := range allResources {
		owner := tracker.FindOwner(ctx, r)

		// if cloudtrail is older than 90 days, find someone who tagged it recenmtly
		if owner == "unknown(>90 days)" {
			if taggedOwner, exists := r.Tags["owner"]; exists {
				owner = taggedOwner + "(via tags)"
			}
		}
		allResources[i].Creator = owner

	}
	// Print summary table
	fmt.Println()
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "RESOURCE ID\tTYPE\tCREATOR\tEST. WASTE")
	fmt.Fprintln(w, "-----------\t----\t-------\t----------")

	for _, r := range allResources {
		fmt.Fprintf(w, "%s\t%s\t%s\t$%.2f\n", r.ID, r.Type, r.Creator, r.EstimatedMo)
		totalWaste += r.EstimatedMo
	}
	w.Flush()

	fmt.Printf("\nTotal Monthly Waste: $%.2f\n\n", totalWaste)

	fmt.Printf("\nTotal monthly waste: $%.2f\n\n", totalWaste)

	if len(allResources) > 0 {
		fmt.Printf("[4/4] generating script for remediation...")
		scriptName := "cleanup.sh"

		err := remediation.CleanupScript(allResources, *region, scriptName)
		if err != nil {
			log.Fatalf("failed to gen script: %v", err)
		}
		fmt.Printf("sucees! Review the script file and run '%s' to safely backup and delete these resources.\n\n", scriptName)
	} else {
		fmt.Printf("No idle resources foudn. your infra is clean")
	}

}
