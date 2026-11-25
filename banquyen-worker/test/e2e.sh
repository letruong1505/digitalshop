#!/bin/bash
# Kiểm thử cục bộ: wrangler dev --local --port 8799 (.dev.vars dùng khóa thử, không phải khóa thật)
B=http://localhost:8799
M="ABCD-EFGH-IJKL-MNOP"
echo "--- tạo đơn sai (thiếu đồng ý)"; curl -s -XPOST $B/api/order -H 'content-type: application/json' -d '{"sp":"fnb","plan":"1y","machine":"'$M'","shop":"Quán A","phone":"0912345678"}'; echo
echo "--- tạo đơn đúng"
R=$(curl -s -XPOST $B/api/order -H 'content-type: application/json' -d '{"sp":"fnb","plan":"1y","machine":"'$M'","shop":"Quán Cà phê Thử <b>","phone":"0912345678","mst":"2902274556","agree":true}')
echo $R
CODE=$(echo $R | python3 -c 'import sys,json;print(json.load(sys.stdin)["code"])'); TOK=$(echo $R | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')
echo "--- trạng thái chờ"; curl -s "$B/api/order?code=$CODE&t=$TOK"; echo
echo "--- sai token"; curl -s "$B/api/order?code=$CODE&t=x"; echo
echo "--- webhook sai key"; curl -s -o /dev/null -w "%{http_code}\n" -XPOST $B/sepay -H 'Authorization: Apikey sai' -d '{}'
W() { curl -s -XPOST $B/sepay -H 'Authorization: Apikey testkey123' -H 'content-type: application/json' -d "$1"; echo; }
echo "--- webhook chuyển thiếu"; W '{"id":1001,"transferType":"in","transferAmount":1000,"accountNumber":"0906027657","content":"CK '$CODE'"}'
curl -s "$B/api/order?code=$CODE&t=$TOK"; echo
echo "--- webhook không khớp đơn"; W '{"id":1002,"transferType":"in","transferAmount":5000,"accountNumber":"0906027657","content":"mua hang"}'
echo "--- webhook sai tài khoản"; W '{"id":1003,"transferType":"in","transferAmount":1500000,"accountNumber":"111","content":"'$CODE'"}'
echo "--- webhook tiền ra"; W '{"id":1004,"transferType":"out","transferAmount":1500000,"accountNumber":"0906027657","content":"'$CODE'"}'
echo "--- webhook đủ tiền (nội dung bị ngân hàng chèn thêm chữ)"; W '{"id":1005,"transferType":"in","transferAmount":1500000,"accountNumber":"0906027657","content":"MBVCB.123.NLBQ-'${CODE#NLBQ}'.CT tu LE VAN A","code":null}'
echo "--- gửi lặp (không cấp lại)"; W '{"id":1005,"transferType":"in","transferAmount":1500000,"accountNumber":"0906027657","content":"'$CODE'"}'
W '{"id":1006,"transferType":"in","transferAmount":1500000,"accountNumber":"0906027657","content":"'$CODE'"}'
echo "--- trạng thái đã cấp"; curl -s "$B/api/order?code=$CODE&t=$TOK" | tee /tmp/bq/paid.json; echo
echo "--- claim theo mã máy"; curl -s "$B/api/claim?sp=fnb&m=abcd-efgh-ijkl-mnop"; echo
echo "--- claim sai sản phẩm"; curl -s "$B/api/claim?sp=shop&m=$M"; echo
echo "--- gia hạn cộng dồn: mua thêm 1 năm"
R=$(curl -s -XPOST $B/api/order -H 'content-type: application/json' -d '{"sp":"fnb","plan":"1y","machine":"'$M'","shop":"Quán A","phone":"0912345678","agree":true}')
C2=$(echo $R | python3 -c 'import sys,json;print(json.load(sys.stdin)["code"])')
W '{"id":1007,"transferType":"in","transferAmount":1500000,"accountNumber":"0906027657","content":"'$C2'"}'
curl -s "$B/api/claim?sp=fnb&m=$M"; echo
echo "--- mua vĩnh viễn rồi mua tiếp bị chặn"
R=$(curl -s -XPOST $B/api/order -H 'content-type: application/json' -d '{"sp":"shop","plan":"life","machine":"'$M'","shop":"Tạp hóa","phone":"0912345678","agree":true}')
C3=$(echo $R | python3 -c 'import sys,json;print(json.load(sys.stdin)["code"])')
W '{"id":1008,"transferType":"in","transferAmount":4500000,"accountNumber":"0906027657","content":"'$C3'"}'
curl -s -XPOST $B/api/order -H 'content-type: application/json' -d '{"sp":"shop","plan":"1y","machine":"'$M'","shop":"Tạp hóa","phone":"0912345678","agree":true}'; echo
echo "--- admin"; curl -s -o /dev/null -w "%{http_code}\n" $B/admin/api/orders; curl -s $B/admin/api/status -H 'Authorization: Bearer adm1n'; echo
curl -s $B/admin/api/orders -H 'Authorization: Bearer adm1n' | python3 -c 'import sys,json;d=json.load(sys.stdin);print(len(d["orders"]),"đơn;",[x["result"] for x in d["odd"]])'
echo "--- duyệt tay"
R=$(curl -s -XPOST $B/api/order -H 'content-type: application/json' -d '{"sp":"fnb","plan":"2y","machine":"WXYZ-2345-6723-4567","shop":"Quán B","phone":"0987654321","agree":true}')
C4=$(echo $R | python3 -c 'import sys,json;print(json.load(sys.stdin)["code"])')
curl -s -XPOST $B/admin/api/approve -H 'Authorization: Bearer adm1n' -d '{"code":"'$C4'","note":"tiền mặt"}'; echo
curl -s "$B/api/claim?sp=fnb&m=WXYZ23456723456 7" ; echo
curl -s "$B/mua" | head -c 200; echo
