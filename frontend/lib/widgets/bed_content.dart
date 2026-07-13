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
  List<dynamic> _beds = [];
  bool _loading = false;

  @override
  void didUpdateWidget(BedContent oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.roomCode != widget.roomCode) {
      _loadBeds();
    }
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
        _beds = List.from(resp['bedsInfo']);
      }
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  bool _isCollected(String bedCode) {
    return widget.collection.any((c) => c['bedCode'] == bedCode);
  }

  void _addToCollection(dynamic bed) {
    final bedMap = bed as Map<String, dynamic>;
    final bedCode = (bedMap['bedCode'] ?? bedMap['code'] ?? '').toString();
    final bedName = (bedMap['bedName'] ?? bedMap['name'] ?? bedCode).toString();
    final newCol = List<Map<String, dynamic>>.from(widget.collection);
    newCol.add({
      'bedCode': bedCode,
      'bedName': bedName,
      'roomCode': widget.roomCode,
      'buildingCode': '',
      'priority': newCol.length + 1,
    });
    widget.onCollectionChanged(newCol, 10);
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) return const Center(child: CircularProgressIndicator());

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.meeting_room_rounded, size: 18, color: primaryColor),
              const SizedBox(width: 8),
              Text('房间 ${widget.roomCode}',
                  style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700, color: textPrimary)),
              const Spacer(),
              IconButton(
                icon: const Icon(Icons.refresh_rounded, size: 20),
                onPressed: _loadBeds,
              ),
            ],
          ),
          const SizedBox(height: 12),
          Expanded(
            child: _beds.isEmpty
                ? const Center(child: Text('该房间暂无可选床位', style: TextStyle(fontSize: 14, color: textMuted)))
                : ListView.builder(
                    itemCount: _beds.length,
                    itemBuilder: (_, i) {
                      final bed = _beds[i] as Map<String, dynamic>;
                      final code = (bed['bedCode'] ?? bed['code'] ?? '').toString();
                      final name = (bed['bedName'] ?? bed['name'] ?? code).toString();
                      final status = (bed['status'] ?? '').toString();
                      final collected = _isCollected(code);
                      final isOccupied = status != '0' && status != '空闲' && status != 'idle';

                      return Container(
                        margin: const EdgeInsets.only(bottom: 8),
                        decoration: BoxDecoration(
                          color: surfaceColor,
                          borderRadius: BorderRadius.circular(radiusLg),
                          border: Border.all(color: borderColor),
                        ),
                        child: Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                          child: Row(
                            children: [
                              Container(
                                width: 3, height: 36,
                                decoration: BoxDecoration(
                                  color: isOccupied ? textMuted : (collected ? successColor : primaryColor),
                                  borderRadius: BorderRadius.circular(2),
                                ),
                              ),
                              const SizedBox(width: 12),
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text(name, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: textPrimary)),
                                    Text(code, style: const TextStyle(fontSize: 11, fontFamily: 'monospace', color: textSecondary)),
                                  ],
                                ),
                              ),
                              if (isOccupied)
                                Container(
                                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                                  decoration: BoxDecoration(
                                    color: textMuted.withAlpha(25),
                                    borderRadius: BorderRadius.circular(radiusSm),
                                  ),
                                  child: const Text('已占', style: TextStyle(fontSize: 12, color: textMuted)),
                                )
                              else if (collected)
                                Container(
                                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                                  decoration: BoxDecoration(
                                    color: successColor.withAlpha(25),
                                    borderRadius: BorderRadius.circular(radiusSm),
                                  ),
                                  child: const Text('已收藏', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: successColor)),
                                )
                              else if (!widget.readOnly)
                                Container(
                                  decoration: BoxDecoration(
                                    gradient: primaryGradient,
                                    borderRadius: BorderRadius.circular(radiusMd),
                                  ),
                                  child: InkWell(
                                    borderRadius: BorderRadius.circular(radiusMd),
                                    onTap: () => _addToCollection(bed),
                                    child: const Padding(
                                      padding: EdgeInsets.symmetric(horizontal: 14, vertical: 6),
                                      child: Text('+ 收藏', style: TextStyle(color: Colors.white, fontSize: 12.5, fontWeight: FontWeight.w600)),
                                    ),
                                  ),
                                ),
                            ],
                          ),
                        ),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}
