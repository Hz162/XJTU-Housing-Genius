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

  String _roomTitle = '';

  Future<void> _loadBeds() async {
    setState(() => _loading = true);
    try {
      final resp = await widget.api.getRoomBeds(widget.divideId, widget.roomCode);
      if (resp['code'] == 0 && resp['bedsInfo'] != null) {
        _rooms = List.from(resp['bedsInfo']);
        // 取第一个 bedsInfo 的 name 作为房间标题（原网页也是这样显示的）
        if (_rooms.isNotEmpty) {
          _roomTitle = ((_rooms[0] as Map<String, dynamic>)['name'] ?? widget.roomCode).toString();
        }
      }
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  bool _isCollected(dynamic room) {
    final code = (room['code'] ?? room['id'] ?? '').toString();
    return widget.collection.any((c) => c['roomCode'] == code);
  }

  /// 点击房间卡片 → 弹出床位选择 Dialog（对齐原网页 "选择床位" 弹窗）
  void _openBedDialog(Map<String, dynamic> roomData) async {
    final roomCode = (roomData['code'] ?? roomData['id'] ?? '').toString();
    final roomName = (roomData['name'] ?? roomCode).toString();
    final beds = (roomData['bedList'] ?? []) as List;
    if (beds.isEmpty) return;

    // 选中态（默认无选中）
    int? selectedIndex;
    for (var i = 0; i < beds.length; i++) {
      if (beds[i]['sn'] == null) { selectedIndex = i; break; }
    }

    final result = await showDialog<Map<String, dynamic>>(
      context: context,
      builder: (ctx) {
        return _BedSelectDialog(
          roomName: roomName,
          beds: beds,
          initialIndex: selectedIndex,
        );
      },
    );

    if (result == null || !mounted) return;

    final bedCodeVal = result['code']?.toString() ?? result['id']?.toString() ?? '';
    final bedName = '$roomName ${result['name'] ?? ''}';
    final allCodes = beds.map((b) => b['id']?.toString() ?? '').where((id) => id.isNotEmpty).join(',');

    // 同步收藏到服务器 (原网页 saveBed API: bedPlaceCode=选中床位的id, bedCodes=全部床id)
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
          Expanded(child: Text(_roomTitle.isNotEmpty ? _roomTitle : '房间 ${widget.roomCode}',
              style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w700, color: textPrimary),
              maxLines: 2, overflow: TextOverflow.ellipsis)),
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
                      child: InkWell(
                        borderRadius: BorderRadius.circular(radiusLg),
                        onTap: (!allFull && !widget.readOnly && !collected)
                            ? () => _openBedDialog(room)
                            : null,
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
                                child: const Padding(
                                  padding: EdgeInsets.symmetric(horizontal: 14, vertical: 6),
                                  child: Text('选择床位', style: TextStyle(color: Colors.white, fontSize: 12.5, fontWeight: FontWeight.w600)),
                                ),
                              ),
                          ]),
                        ),
                      ),
                    );
                  },
                ),
        ),
      ]),
    );
  }
}

/// 床位选择弹窗 —— 对齐原网页 Dialog "选择床位" + sqPreviewGroup
class _BedSelectDialog extends StatefulWidget {
  final String roomName;
  final List beds;
  final int? initialIndex;

  const _BedSelectDialog({
    required this.roomName,
    required this.beds,
    required this.initialIndex,
  });

  @override
  State<_BedSelectDialog> createState() => _BedSelectDialogState();
}

class _BedSelectDialogState extends State<_BedSelectDialog> {
  late int _selected;

  @override
  void initState() {
    super.initState();
    _selected = widget.initialIndex ?? 0;
  }

  @override
  Widget build(BuildContext context) {
    final freeCount = widget.beds.where((b) => b['sn'] == null).length;

    return AlertDialog(
      title: const Text('选择床位', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
      content: SizedBox(
        width: 320,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(widget.roomName,
                style: const TextStyle(fontSize: 13, color: textSecondary)),
            const SizedBox(height: 4),
            Text('共${widget.beds.length}个床位 · $freeCount个空闲',
                style: const TextStyle(fontSize: 11, color: textMuted)),
            const SizedBox(height: 16),
            // 床位选择标签 —— 对齐原网页 sq-preview-item chips
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: List.generate(widget.beds.length, (i) {
                final bed = widget.beds[i] as Map<String, dynamic>;
                final isTaken = bed['sn'] != null;
                final isSelected = _selected == i;

                return GestureDetector(
                  onTap: isTaken
                      ? null
                      : () => setState(() => _selected = i),
                  child: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
                    decoration: BoxDecoration(
                      color: isTaken
                          ? const Color(0xFFF2F3F5)
                          : isSelected
                              ? const Color(0xFFFEE0E3)
                              : const Color(0xFFF5F5F5),
                      borderRadius: BorderRadius.circular(6),
                      border: isSelected && !isTaken
                          ? Border.all(color: dangerColor, width: 1.5)
                          : null,
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text(
                          bed['name']?.toString() ?? '${i + 1}号床',
                          style: TextStyle(
                            fontSize: 13,
                            fontWeight: isSelected ? FontWeight.w600 : FontWeight.w400,
                            color: isTaken
                                ? const Color(0xFFC8C9CC)
                                : isSelected
                                    ? dangerColor
                                    : const Color(0xFF323233),
                          ),
                        ),
                        if (isTaken) ...[
                          const SizedBox(width: 4),
                          const Text('已选', style: TextStyle(fontSize: 10, color: Color(0xFFC8C9CC))),
                        ],
                      ],
                    ),
                  ),
                );
              }),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('取消', style: TextStyle(color: textSecondary)),
        ),
        FilledButton(
          onPressed: freeCount == 0
              ? null
              : () {
                  final bed = widget.beds[_selected] as Map<String, dynamic>;
                  if (bed['sn'] != null) return; // 安全：不收藏已选的
                  Navigator.pop(context, bed);
                },
          style: FilledButton.styleFrom(
            backgroundColor: primaryColor,
          ),
          child: const Text('收藏', style: TextStyle(color: Colors.white)),
        ),
      ],
    );
  }
}
