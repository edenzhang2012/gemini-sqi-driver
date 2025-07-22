package pkg

import (
	"testing"
	"time"

	"github.com/edenzhang2012/storagequotainterface/sqi/pb"
)

func TestGetPluginCapabilities(t *testing.T) {
	service, err := NewStorageQuotaPluginService()
	if err != nil {
		t.Errorf("NewStorageQuotaPluginService failed with %v", err)
		return
	}
	res, err := service.GetPluginCapabilities(t.Context(), &pb.GetPluginCapabilitiesRequest{})
	if err != nil {
		t.Errorf("GetPluginCapabilities failed with %v", err)
		return
	}

	mustRpc := 0

	for _, cap := range res.Capabilities {
		switch v := cap.Capability.(type) {
		case *pb.PluginCapability_Rpc:
			if v.Rpc == pb.PluginCapability_UNKNOWN_RPC || v.Rpc > pb.PluginCapability_TARGET_INFO {
				t.Errorf("unsupport PluginCapability_Rpc %v", v.Rpc)
			}
			if v.Rpc > pb.PluginCapability_UNKNOWN_RPC && v.Rpc < pb.PluginCapability_VALIDATE_QUOTA {
				mustRpc++
			}
		case *pb.PluginCapability_Quota:
			if v.Quota == pb.PluginCapability_UNKNOWN_QUOTA || v.Quota > pb.PluginCapability_INODES {
				t.Errorf("unsupport PluginCapability_Quota %v", v.Quota)
			}
		case *pb.PluginCapability_Id:
			if v.Id == pb.PluginCapability_UNKNOWN_ID || v.Id > pb.PluginCapability_VENDOR {
				t.Errorf("unsupport PluginCapability_Quota %v", v.Id)
			}
		default:
			t.Errorf("unsupport Capability type %v", v)
		}
	}

	if mustRpc < 4 {
		t.Errorf("PluginCapability_Rpc must > 5")
	}
}

func TestQuotaCRUD(t *testing.T) {
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

	service, err := NewStorageQuotaPluginService()
	if err != nil {
		t.Errorf("NewStorageQuotaPluginService failed with %v", err)
		return
	}

	//test create
	for _, test := range testCases {
		//set
		setRes, err := service.SetQuota(t.Context(), &pb.SetQuotaRequest{
			Target: &pb.QuotaTarget{
				Scope: test.req.Scope,
				Id:    test.req.Id,
			},
			SizeBytes: test.req.SizeBytes,
		})
		if err != nil {
			t.Errorf("SetQuota failed with %v", err)
		} else {
			t.Logf("set quota success res %v", setRes.Info)

			//get
			for {
				getRes, err := service.GetQuota(t.Context(), &pb.GetQuotaRequest{
					Target: &pb.QuotaTarget{
						Scope: test.req.Scope,
						Id:    test.req.Id,
					},
				})
				if err != nil {
					t.Errorf("GetQuota failed with %v", err)
				} else {
					//status == ok or no status means all good
					if valve, ok := getRes.Entry.Info["status"]; ok {
						if valve != "ok" {
							time.Sleep(500 * time.Millisecond)
							continue
						}
					}
					if getRes.Entry.SizeQuotaEnable != test.expected.SizeQuotaEnable {
						t.Errorf("getRes.Entries.SizeQuotaEnable is %v, expact %v", getRes.Entry.SizeQuotaEnable, test.expected.SizeQuotaEnable)
					}
					if getRes.Entry.InodeQuotaEnable != test.expected.InodeQuotaEnable {
						t.Errorf("getRes.Entries.InodeQuotaEnable is %v, expact %v", getRes.Entry.InodeQuotaEnable, test.expected.InodeQuotaEnable)
					}
					if getRes.Entry.SizeBytes != test.expected.SizeBytes {
						t.Errorf("getRes.Entries.SizeBytes is %d, expact %d", getRes.Entry.SizeBytes, test.expected.SizeBytes)
					}

					t.Logf("getRes %v", getRes.Entry)
					t.Logf("used %d", getRes.Entry.UsedBytes)
					break
				}
			}
		}
	}

	//list with limit
	listRes, err := service.ListQuotas(t.Context(), &pb.ListQuotasRequest{
		Limit:         5,
		ContinueToken: "",
		Target: &pb.QuotaTarget{
			Scope: pb.QuotaTarget_PATH,
			Id:    "",
		},
	})
	if err != nil {
		t.Errorf("ListQuotas failed with %v", err)
	} else {
		t.Logf("ListQuotas get reault %v", listRes)
		if len(listRes.Entries) != 5 {
			t.Errorf("ListQuotas get %d entries, expact %d", len(listRes.Entries), 5)
		}
	}

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
		_, err := service.ClearQuota(t.Context(), &pb.ClearQuotaRequest{
			Target: &pb.QuotaTarget{
				Scope: test.scope,
				Id:    test.id,
			},
		})
		if err != nil {
			t.Errorf("ClearQuota failed with %v", err)
		}
	}

	//list
	listRes, err = service.ListQuotas(t.Context(), &pb.ListQuotasRequest{
		Limit:         5,
		ContinueToken: "",
		Target: &pb.QuotaTarget{
			Scope: pb.QuotaTarget_PATH,
			Id:    "",
		},
	})
	if err != nil {
		t.Errorf("ListQuotas failed with %v", err)
	} else {
		t.Logf("ListQuotas get reault %v", listRes)
		if len(listRes.Entries) != 0 {
			t.Errorf("ListQuotas get %d entries, expact %d", len(listRes.Entries), 0)
		}
	}
}
