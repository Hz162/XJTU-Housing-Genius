import 'package:flutter/material.dart';
import '../services/api_service.dart';
import '../theme/app_theme.dart';

class BedContent extends StatefulWidget {
  final ApiService api;
  final String divideId;
  final String roomCode;
  final String personsn;
  final List<Map<String, dynamic>> collection;
  final bool readOnly;
  final void Function(List<Map<String, dynamic>>, int) onCollectionChanged;

  const BedContent({
    super.key,
    required this.api,
    required this.divideId,
    required this.roomCode,
    required this.personsn,
    required this.collection,
    required this.readOnly,
    required this.onCollectionChanged,
  });

  @override
  State<BedContent> createState() => _BedContentState();
}

class _BedContentState extends State<BedContent> {
  List<dynamic> _rooms = [];
  bool _loading = false;

  @override
  void didUpdateWidget(BedContent oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.roomCode != widget.roomCode) _loadBeds();
  }

  @override
  void initState() {
    super.initState();
    _loadBeds();
  }

  Future<void> _loadBeds() async {
    setState(() => _loading = true);
    try {
      final resp = await widget.api.getRoomBeds(widget.divideId, widget.roomCode);
      if (resp['code'] == 0 && resp['bedsInfo'] != null) {
        _rooms = List.from(resp['bedsInfo']);
      }
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  bool _isCollected(dynamic room) {
    final code = (room['code'] ?? room['id'] ?? '').toString();
    return widget.collection.any((c) => c['roomCode'] == code);
  }

  void _addRoom(dynamic roomData) async {
    final room = roomData as Map<String, dynamic>;
    final roomCode = (room['code'] ?? room['id'] ?? '').toString();
    final beds = (room['bedList'] ?? []) as List;
    if (beds.isEmpty) return;
    Map<String, dynamic>? pick;
    for (final b in beds) {
      if (b['sn'] == null) { pick = Map<String, dynamic>.from(b); break; }
    }
    pick ??= Map<String, dynamic>.from(beds.first);
    final bedCodeVal = pick['code']?.toString() ?? pick['id']?.toString() ?? '';
    final bedName = '${room['name'] ?? roomCode} ${pick['name'] ?? ''}';
    final allCodes = beds.map((b) => b['id']?.toString() ?? '').where((id) => id.isNotEmpty).join(',');

    // 同步收藏到服务器 (原网页 saveBed API)
    try {
      await widget.api.collectSyncSave(
        personsn: widget.personsn,
        bedPlaceCode: bedCodeVal,
        divideId: widget.divideId,
        bedCodes: allCodes,
      );
    } catch (_) {}

    final newCol = List<Map<String, dynamic>>.from(widget.collection);
    newCol.add({
      'bedCode': bedCodeVal, 'bedName': bedName,
      'roomCode': roomCode, 'buildingCode': '',
      'priority': newCol.length + 1, 'bedCodes': allCodes,
    });
    widget.onCollectionChanged(newCol, 10);
    if (mounted) setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) return const Center(child: CircularProgressIndicator());
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          const Icon(Icons.meeting_room_rounded, size: 18, color: primaryColor),
          const SizedBox(width: 8),
          Expanded(child: Text('房间 ${widget.roomCode}',
              style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700, color: textPrimary))),
          IconButton(icon: const Icon(Icons.refresh_rounded, size: 20), onPressed: _loadBeds),
        ]),
        const SizedBox(height: 12),
        Expanded(
          child: _rooms.isEmpty
              ? Center(
                  child: GestureDetector(
                    onTap: _loadBeds,
                    child: const Text('加载失败，点击重试', style: TextStyle(fontSize: 14, color: textMuted)),
                  ))
              : ListView.builder(
                  itemCount: _rooms.length,
                  itemBuilder: (_, i) {
                    final room = _rooms[i] as Map<String, dynamic>;
                    final code = (room['code'] ?? room['id'] ?? '').toString();
                    final name = (room['name'] ?? code).toString();
                    final beds = (room['bedList'] ?? []) as List;
                    final freeBeds = beds.where((b) => b['sn'] == null).length;
                    final hasBadge = room['badge'] == true;
                    final collected = _isCollected(room);
                    final allFull = freeBeds == 0;

                    return Container(
                      margin: const EdgeInsets.only(bottom: 8),
                      decoration: BoxDecoration(
                        color: surfaceColor,
                        borderRadius: BorderRadius.circular(radiusLg),
                        border: Border.all(color: borderColor),
                      ),
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                        child: Row(children: [
                          Container(width: 3, height: 36, decoration: BoxDecoration(
                            color: allFull ? textMuted : (collected ? successColor : primaryColor),
                            borderRadius: BorderRadius.circular(2),
                          )),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                              Text(name, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: textPrimary)),
                              Text('${beds.length}个床位 · 空闲$freeBeds', style: const TextStyle(fontSize: 11, color: textSecondary)),
                            ]),
                          ),
                          if (hasBadge)
                            Container(
                              margin: const EdgeInsets.only(right: 6),
                              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                              decoration: BoxDecoration(color: dangerColor.withAlpha(20), borderRadius: BorderRadius.circular(4)),
                              child: const Text('满', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w600, color: dangerColor)),
                            ),
                          if (allFull)
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                              decoration: BoxDecoration(color: textMuted.withAlpha(25), borderRadius: BorderRadius.circular(radiusSm)),
                              child: const Text('已满', style: TextStyle(fontSize: 12, color: textMuted)),
                            )
                          else if (collected)
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                              decoration: BoxDecoration(color: successColor.withAlpha(25), borderRadius: BorderRadius.circular(radiusSm)),
                              child: const Text('已收藏', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: successColor)),
                            )
                          else if (!widget.readOnly)
                            Container(
                              decoration: BoxDecoration(gradient: primaryGradient, borderRadius: BorderRadius.circular(radiusMd)),
                              child: InkWell(
                                borderRadius: BorderRadius.circular(radiusMd),
                                onTap: () => _addRoom(room),
                                child: const Padding(
                                  padding: EdgeInsets.symmetric(horizontal: 14, vertical: 6),
                                  child: Text('+ 收藏', style: TextStyle(color: Colors.white, fontSize: 12.5, fontWeight: FontWeight.w600)),
                                ),
                              ),
                            ),
                        ]),
                      ),
                    );
                  },
                ),
        ),
      ]),
    );
  }
}
