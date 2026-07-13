package bed

import "encoding/json"

// MockMode 模拟模式（真实 housing 不可用时启用）
var MockMode = false

// ── 模拟楼栋树 ──

func MockBunkTree() []byte {
	tree := []map[string]any{
		{
			"code": "B1", "name": "B1栋（兴庆校区）", "text": "B1栋（兴庆校区）",
			"children": []map[string]any{
				{
					"code": "B1-F1", "name": "1层", "text": "1层",
					"children": []map[string]any{
						{"code": "B1-101", "name": "101室", "text": "101室", "roomCode": "B1-101"},
						{"code": "B1-102", "name": "102室", "text": "102室", "roomCode": "B1-102"},
						{"code": "B1-103", "name": "103室", "text": "103室", "roomCode": "B1-103"},
					},
				},
				{
					"code": "B1-F2", "name": "2层", "text": "2层",
					"children": []map[string]any{
						{"code": "B1-201", "name": "201室", "text": "201室", "roomCode": "B1-201"},
						{"code": "B1-202", "name": "202室", "text": "202室", "roomCode": "B1-202"},
					},
				},
			},
		},
		{
			"code": "B2", "name": "B2栋（兴庆校区）", "text": "B2栋（兴庆校区）",
			"children": []map[string]any{
				{
					"code": "B2-F1", "name": "1层", "text": "1层",
					"children": []map[string]any{
						{"code": "B2-101", "name": "101室", "text": "101室", "roomCode": "B2-101"},
						{"code": "B2-102", "name": "102室", "text": "102室", "roomCode": "B2-102"},
					},
				},
				{
					"code": "B2-F2", "name": "2层", "text": "2层",
					"children": []map[string]any{
						{"code": "B2-201", "name": "201室", "text": "201室", "roomCode": "B2-201"},
					},
				},
			},
		},
		{
			"code": "B3", "name": "B3栋（雁塔校区）", "text": "B3栋（雁塔校区）",
			"children": []map[string]any{
				{
					"code": "B3-F1", "name": "1层", "text": "1层",
					"children": []map[string]any{
						{"code": "B3-101", "name": "101室", "text": "101室", "roomCode": "B3-101"},
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
	beds := []map[string]any{
		{"bedCode": roomCode + "-A", "bedName": roomCode + " A床（靠窗）", "code": roomCode + "-A", "name": "A床（靠窗）", "status": "0"},
		{"bedCode": roomCode + "-B", "bedName": roomCode + " B床（靠门）", "code": roomCode + "-B", "name": "B床（靠门）", "status": "0"},
		{"bedCode": roomCode + "-C", "bedName": roomCode + " C床（中间）", "code": roomCode + "-C", "name": "C床（中间）", "status": "1"},
		{"bedCode": roomCode + "-D", "bedName": roomCode + " D床（靠窗）", "code": roomCode + "-D", "name": "D床（靠窗）", "status": "0"},
	}
	resp := map[string]any{"code": 0, "bedsInfo": beds}
	b, _ := json.Marshal(resp)
	return b
}

// ── 模拟 divideId ──

func MockDivideId() []byte {
	resp := map[string]any{
		"code": 0,
		"map": map[string]any{
			"divideId": "MOCK-DIVIDE-2026",
			"disabled": false,
			"time":     "2026-07-13 12:00:00",
			"endtime":  "2026-07-15 23:59:59",
		},
		"divideId": "MOCK-DIVIDE-2026",
	}
	b, _ := json.Marshal(resp)
	return b
}

// ── 模拟 checkMyBed ──

func MockCheckMyBed() []byte {
	resp := map[string]any{"code": 0, "isMybed": false}
	b, _ := json.Marshal(resp)
	return b
}

// ── 模拟 distributeBed ──

var mockGrabAttempt int

func MockDistributeBed() []byte {
	mockGrabAttempt++
	// 模拟：前 2 次失败，第 3 次成功
	if mockGrabAttempt >= 3 {
		return []byte(`{"code":0,"status":0,"promptMsg":"选床成功！恭喜抢到床位！"}`)
	}
	return []byte(`{"code":0,"status":0,"promptMsg":"床位已被其他人抢走"}`)
}

func ResetMockGrab() { mockGrabAttempt = 0 }
