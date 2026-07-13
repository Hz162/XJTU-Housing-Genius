package bed

import (
	"encoding/json"
	"fmt"
)

var MockMode = false

// mockRoomNames maps room code → display name for MockRoomBeds
var mockRoomNames = map[string]string{
	"020201": "创新港校区-和园（A区）-A02幢-二单元-2-020201",
	"020202": "创新港校区-和园（A区）-A02幢-二单元-2-020202",
	"020301": "创新港校区-和园（A区）-A02幢-二单元-3-020301",
	"020302": "创新港校区-和园（A区）-A02幢-二单元-3-020302",
	"101":    "兴庆校区-东区-东1楼-1单元-1-101",
	"102":    "兴庆校区-东区-东1楼-1单元-1-102",
}

// ── 模拟楼栋树 (7层: ROOT → CAMPUS → PARK → BUILDING → UNIT → FLOOR → ROOM) ──

func MockBunkTree() []byte {
	tree := []map[string]any{
		{
			"label": "西安交通大学", "type": "ROOT", "value": "root",
			"children": []map[string]any{
				{
					"label": "创新港校区", "type": "CAMPUS", "value": "campus1",
					"children": []map[string]any{
						{
							"label": "和园（A区）", "type": "PARK", "value": "park1",
							"children": []map[string]any{
								{
									"label": "A02幢", "type": "BUILDING", "value": "build_a02",
									"children": []map[string]any{
										{
											"label": "二单元", "type": "UNIT", "value": "unit_a02_2",
											"children": []map[string]any{
												{
													"label": "2", "type": "FLOOR", "value": "floor_a02_2_2",
													"floorUrl": "",
													"children": []map[string]any{
														{"label": "020201", "type": "ROOM", "value": "020201"},
														{"label": "020202", "type": "ROOM", "value": "020202"},
													},
												},
												{
													"label": "3", "type": "FLOOR", "value": "floor_a02_2_3",
													"floorUrl": "",
													"children": []map[string]any{
														{"label": "020301", "type": "ROOM", "value": "020301"},
														{"label": "020302", "type": "ROOM", "value": "020302"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				{
					"label": "兴庆校区", "type": "CAMPUS", "value": "campus2",
					"children": []map[string]any{
						{
							"label": "东区", "type": "PARK", "value": "park2",
							"children": []map[string]any{
								{
									"label": "东1楼", "type": "BUILDING", "value": "build_e1",
									"children": []map[string]any{
										{
											"label": "1单元", "type": "UNIT", "value": "unit_e1_1",
											"children": []map[string]any{
												{
													"label": "1", "type": "FLOOR", "value": "floor_e1_1",
													"floorUrl": "",
													"children": []map[string]any{
														{"label": "101", "type": "ROOM", "value": "101"},
														{"label": "102", "type": "ROOM", "value": "102"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	b, _ := json.Marshal(tree)
	return b
}

// ── 模拟房间床位 ──

func MockRoomBeds(roomCode string) []byte {
	roomName := mockRoomNames[roomCode]
	if roomName == "" {
		roomName = fmt.Sprintf("房间-%s", roomCode)
	}
	bedNames := []string{"1号床", "2号床", "3号床", "4号床"}
	bedList := make([]map[string]any, len(bedNames))
	for i, name := range bedNames {
		sn := any(nil)
		if i >= 2 { // 第3、4床已被选
			sn = "3125303000"
		}
		bedList[i] = map[string]any{
			"id":   fmt.Sprintf("%s-%d", roomCode, i+1),
			"code": fmt.Sprintf("%s-%d", roomCode, i+1),
			"name": name,
			"sn":   sn,
		}
	}
	resp := map[string]any{
		"code": 0,
		"bedsInfo": []map[string]any{
			{
				"code":    roomCode,
				"name":    roomName,
				"badge":   false,
				"roomUrl": "",
				"bedList": bedList,
			},
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

// ── 模拟其他 API ──

func MockDivideId() []byte {
	resp := map[string]any{
		"code": 0,
		"divideCountDown": map[string]any{
			"id":       "MOCK-DIVIDE-2026",
			"disabled": false,
			"time":     "2026-07-13 12:00:00",
			"endtime":  "2026-07-15 23:59:59",
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

func MockCheckMyBed() []byte {
	resp := map[string]any{"code": 0, "isMybed": false}
	b, _ := json.Marshal(resp)
	return b
}

func MockCollectList() []byte {
	resp := map[string]any{
		"code": 0,
		"bedCollects": []map[string]any{
			{
				"id":          "mock-collect-1",
				"code":        "020201-2",
				"name":        "创新港校区-和园（A区）-A02幢-二单元-2-020201",
				"bedName":     "2号床",
				"url":         "",
				"status":      "0",
				"num":         3,
				"beddingInfo": "[]",
				"bedCodes":    "020201-1,020201-2,020201-3,020201-4",
			},
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

var mockGrabAttempt int

func MockDistributeBed() []byte {
	mockGrabAttempt++
	if mockGrabAttempt >= 3 {
		return []byte(`{"code":0,"status":0,"promptMsg":"选床成功！"}`)
	}
	return []byte(`{"code":0,"status":1,"promptMsg":"床位已被抢，重试中..."}`)
}

func ResetMockGrab() { mockGrabAttempt = 0 }
