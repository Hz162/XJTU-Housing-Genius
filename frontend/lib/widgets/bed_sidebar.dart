import 'package:flutter/material.dart';
import '../services/api_service.dart';
import '../theme/app_theme.dart';

class BedSidebar extends StatefulWidget {
  final bool collapsed;
  final ApiService api;
  final String divideId;
  final void Function(String roomCode) onRoomSelected;

  const BedSidebar({
    super.key,
    required this.collapsed,
    required this.api,
    required this.divideId,
    required this.onRoomSelected,
  });

  @override
  State<BedSidebar> createState() => _BedSidebarState();
}

class _BedSidebarState extends State<BedSidebar> {
  List<dynamic> _treeData = [];
  bool _loading = true;
  String? _selectedRoom;

  @override
  void didUpdateWidget(BedSidebar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.divideId != widget.divideId && widget.divideId.isNotEmpty) {
      _loadTree();
    }
  }

  @override
  void initState() {
    super.initState();
    if (widget.divideId.isNotEmpty) _loadTree();
  }

  Future<void> _loadTree() async {
    setState(() => _loading = true);
    try {
      _treeData = await widget.api.getBedTree(widget.divideId);
    } catch (_) {}
    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    if (widget.collapsed) return const SizedBox(width: 0);

    return Material(
      color: surfaceColor,
      child: Container(
        width: 240,
        decoration: const BoxDecoration(
          border: Border(right: BorderSide(color: borderColor)),
        ),
        child: _loading
          ? const Center(child: CircularProgressIndicator(strokeWidth: 2))
          : ListView(
              padding: const EdgeInsets.all(8),
              children: _treeData.map((building) => _buildBuilding(building)).toList(),
            ),
    ),
    );
  }

  Widget _buildBuilding(dynamic building) {
    final map = building as Map<String, dynamic>;
    final name = (map['name'] ?? map['label'] ?? map['buildingName'] ?? map['text'] ?? '?').toString();
    final children = (map['children'] ?? map['floors'] ?? map['nodes'] ?? []) as List;
    return ExpansionTile(
      tilePadding: const EdgeInsets.symmetric(horizontal: 8),
      childrenPadding: const EdgeInsets.only(left: 16),
      leading: const Icon(Icons.apartment_rounded, size: 18, color: primaryColor),
      title: Text(name,
          style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: textPrimary)),
      children: children.map((floor) => _buildFloor(floor as Map<String, dynamic>)).toList(),
    );
  }

  Widget _buildFloor(Map<String, dynamic> floor) {
    final name = (floor['name'] ?? floor['floorName'] ?? floor['text'] ?? '?').toString();
    final children = (floor['children'] ?? floor['rooms'] ?? floor['nodes'] ?? []) as List;
    return ExpansionTile(
      tilePadding: const EdgeInsets.symmetric(horizontal: 8),
      leading: const Icon(Icons.layers_rounded, size: 16, color: textSecondary),
      title: Text(name,
          style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w500, color: textSecondary)),
      children: children.map((room) => _buildRoom(room as Map<String, dynamic>)).toList(),
    );
  }

  Widget _buildRoom(Map<String, dynamic> room) {
    final name = (room['name'] ?? room['roomName'] ?? room['text'] ?? '?').toString();
    final code = (room['code'] ?? room['roomCode'] ?? room['id'] ?? '').toString();
    final isSelected = _selectedRoom == code;
    return ListTile(
      dense: true,
      selected: isSelected,
      selectedTileColor: primaryColor.withAlpha(20),
      leading: Icon(Icons.meeting_room_rounded,
          size: 14, color: isSelected ? primaryColor : textMuted),
      title: Text(name,
          style: TextStyle(
              fontSize: 12,
              fontWeight: isSelected ? FontWeight.w600 : FontWeight.w400,
              color: isSelected ? primaryColor : textSecondary)),
      onTap: () {
        setState(() => _selectedRoom = code);
        widget.onRoomSelected(code);
      },
    );
  }
}
