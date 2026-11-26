package main

import "testing"

func TestFeedIngest(t *testing.T) {
	feedReset(); feedEvts = nil; feedSeq = 0
	old := []sepayTx{{ID: "2", In: "20000.00", Content: "cu 2"}, {ID: "1", In: "10000.00", Content: "cu 1"}}
	feedIngest(old) // lần đầu: chỉ ghi nhớ
	if feedSeq != 0 { t.Fatalf("lan dau khong duoc phat, seq=%d", feedSeq) }
	feedIngest(append([]sepayTx{{ID: "4", In: "0.00"}, {ID: "3", In: "15000.00", Content: "NLBQ"}}, old...))
	if feedSeq != 1 || feedEvts[0].Amount != 15000 { t.Fatalf("seq=%d evts=%v", feedSeq, feedEvts) }
	feedIngest(append([]sepayTx{{ID: "3", In: "15000.00"}}, old...)) // lặp: không phát lại
	if feedSeq != 1 { t.Fatalf("phat lai: %d", feedSeq) }
}
