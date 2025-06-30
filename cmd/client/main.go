package main

import (
	"context"
	"log"
	"time"

	"github.com/edenzhang2012/storagequotainterface/sqi/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	AppName = "storagequotaplugin"
)

func main() {
	//grpc client初始化
	conn, err := grpc.NewClient("unix:/var/run/"+AppName+"/"+AppName+".sock", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewQuotaServiceClient(conn)

	//client demo： simple tests for all client interfaces
	for {
		//GetPluginCapabilities
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		res, err := c.GetPluginCapabilities(ctx, &pb.GetPluginCapabilitiesRequest{})
		if err != nil {
			log.Fatalf("GetPluginCapabilities failed with %v", err)
		} else {
			log.Printf("Capabilities: %v", res.Capabilities)

			mustRpc := 0

			for _, cap := range res.Capabilities {
				switch v := cap.Capability.(type) {
				case *pb.PluginCapability_Rpc:
					if v.Rpc == pb.PluginCapability_UNKNOWN_RPC || v.Rpc > pb.PluginCapability_TARGET_INFO {
						log.Fatalf("unsupport PluginCapability_Rpc %v", v.Rpc)
					}
					if v.Rpc > pb.PluginCapability_UNKNOWN_RPC && v.Rpc < pb.PluginCapability_VALIDATE_QUOTA {
						mustRpc++
					}
				case *pb.PluginCapability_Quota:
					if v.Quota == pb.PluginCapability_UNKNOWN_QUOTA || v.Quota > pb.PluginCapability_INODES {
						log.Fatalf("unsupport PluginCapability_Quota %v", v.Quota)
					}
				case *pb.PluginCapability_Id:
					if v.Id == pb.PluginCapability_UNKNOWN_ID || v.Id > pb.PluginCapability_VENDOR {
						log.Fatalf("unsupport PluginCapability_Quota %v", v.Id)
					}
				default:
					log.Fatalf("unsupport Capability type %v", v)
				}
			}

			if mustRpc < 4 {
				log.Fatalf("PluginCapability_Rpc must > 5")
			}
		}
		cancel()

		//set and get
		type Req struct {
			Scope     pb.QuotaTarget_Scope
			Id        string
			SizeBytes uint64
		}

		type Expected struct {
			SizeQuotaEnable  bool
			InodeQuotaEnable bool
			SizeBytes        uint64
		}

		testCases := []struct {
			req      Req
			expected Expected
		}{
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent", uint64(1024 * 1024 * 1024)}, Expected{true, false, uint64(1024 * 1024 * 1024)}},        //parent
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child1", uint64(100 * 1024 * 1024)}, Expected{true, false, uint64(100 * 1024 * 1024)}},   //child1
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child2", uint64(500 * 1024 * 1024)}, Expected{true, false, uint64(500 * 1024 * 1024)}},   //child2
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child3", uint64(1024 * 1024 * 1024)}, Expected{true, false, uint64(800 * 1024 * 1024)}},  //child3
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child4", uint64(100 * 1024 * 1024)}, Expected{true, false, uint64(100 * 1024 * 1024)}},   //child4
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child5", uint64(500 * 1024 * 1024)}, Expected{true, false, uint64(500 * 1024 * 1024)}},   //child5
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child6", uint64(1024 * 1024 * 1024)}, Expected{true, false, uint64(800 * 1024 * 1024)}},  //child6
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child7", uint64(100 * 1024 * 1024)}, Expected{true, false, uint64(100 * 1024 * 1024)}},   //child7
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child8", uint64(500 * 1024 * 1024)}, Expected{true, false, uint64(500 * 1024 * 1024)}},   //child8
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child9", uint64(1024 * 1024 * 1024)}, Expected{true, false, uint64(800 * 1024 * 1024)}},  //child9
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child10", uint64(100 * 1024 * 1024)}, Expected{true, false, uint64(100 * 1024 * 1024)}},  //child10
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child11", uint64(500 * 1024 * 1024)}, Expected{true, false, uint64(500 * 1024 * 1024)}},  //child11
			{Req{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child12", uint64(1024 * 1024 * 1024)}, Expected{true, false, uint64(800 * 1024 * 1024)}}, //child12
		}

		//test create
		for _, test := range testCases {
			//set
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			setRes, err := c.SetQuota(ctx, &pb.SetQuotaRequest{
				Target: &pb.QuotaTarget{
					Scope: test.req.Scope,
					Id:    test.req.Id,
				},
				SizeBytes: test.req.SizeBytes,
			})
			if err != nil {
				log.Fatalf("SetQuota failed with %v", err)
			} else {
				cancel()
				log.Printf("set quota success res %v", setRes.Info)

				//get
				for {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second)
					getRes, err := c.GetQuota(ctx, &pb.GetQuotaRequest{
						Target: &pb.QuotaTarget{
							Scope: test.req.Scope,
							Id:    test.req.Id,
						},
					})
					if err != nil {
						log.Fatalf("GetQuota failed with %v", err)
					} else {
						//status == ok or no status means all good
						if valve, ok := getRes.Entries.Info["status"]; ok {
							if valve != "ok" {
								time.Sleep(500 * time.Millisecond)
								cancel()
								continue
							}
						}
						if getRes.Entries.SizeQuotaEnable != test.expected.SizeQuotaEnable {
							log.Fatalf("getRes.Entries.SizeQuotaEnable is %v, expact %v", getRes.Entries.SizeQuotaEnable, test.expected.SizeQuotaEnable)
						}
						if getRes.Entries.InodeQuotaEnable != test.expected.InodeQuotaEnable {
							log.Fatalf("getRes.Entries.InodeQuotaEnable is %v, expact %v", getRes.Entries.InodeQuotaEnable, test.expected.InodeQuotaEnable)
						}
						if getRes.Entries.SizeBytes != test.expected.SizeBytes {
							log.Fatalf("getRes.Entries.SizeBytes is %d, expact %d", getRes.Entries.SizeBytes, test.expected.SizeBytes)
						}

						log.Printf("getRes %v", getRes.Entries)
						log.Printf("used %d", getRes.Entries.UsedBytes)
						cancel()
						break
					}
				}
			}
		}

		//list with limit
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
		listRes, err := c.ListQuotas(ctx, &pb.ListQuotasRequest{
			Limit:         5,
			ContinueToken: "",
			FilterScope:   pb.QuotaTarget_PATH,
		})
		if err != nil {
			log.Fatalf("ListQuotas failed with %v", err)
		} else {
			log.Printf("ListQuotas get reault %v", listRes)
			if len(listRes.Entries) != 5 {
				log.Fatalf("ListQuotas get %d entries, expact %d", len(listRes.Entries), 5)
			}
		}
		cancel()

		//clear
		clearCases := []struct {
			scope pb.QuotaTarget_Scope
			id    string
		}{
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child1"},
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child2"},
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child3"},
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child4"},
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child5"},
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child6"},
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child7"},
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child8"},
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child9"},
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child10"},
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child11"},
			{pb.QuotaTarget_PATH, "test-1:/tmp/a/parent/child12"},
		}
		for _, test := range clearCases {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			_, err := c.ClearQuota(ctx, &pb.ClearQuotaRequest{
				Target: &pb.QuotaTarget{
					Scope: test.scope,
					Id:    test.id,
				},
			})
			if err != nil {
				log.Fatalf("ClearQuota failed with %v", err)
			}
			cancel()
		}

		//list
		ctx, cancel = context.WithTimeout(context.Background(), time.Second)
		listRes, err = c.ListQuotas(ctx, &pb.ListQuotasRequest{
			Limit:         5,
			ContinueToken: "",
			FilterScope:   pb.QuotaTarget_PATH,
		})
		if err != nil {
			log.Fatalf("ListQuotas failed with %v", err)
		} else {
			log.Printf("ListQuotas get reault %v", listRes)
			if len(listRes.Entries) != 0 {
				log.Fatalf("ListQuotas get %d entries, expact %d", len(listRes.Entries), 0)
			}
		}
		cancel()

		log.Println("all tests are passed!")
		time.Sleep(5 * time.Second)
	}
}
