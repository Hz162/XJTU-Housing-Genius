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
    if (oldWidget.divideId != widget.divideId) {
      if (widget.divideId.isNotEmpty) {
        _loadTree();
      } else {
        setState(() => _loading = false);
      }
    }
  }

  @override
  void initState() {
    super.initState();
    if (widget.divideId.isNotEmpty) {
      _loadTree();
    } else {
      _loading = false;
    }
  }

  Future<void> _loadTree() async {
    setState(() => _loading = true);
    try {
      _treeData = await widget.api.getBedTree(widget.divideId);
      if (_treeData.isEmpty && mounted) {
        await Future.delayed(const Duration(seconds: 2));
        _treeData = await widget.api.getBedTree(widget.divideId);
      }
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
            ? GestureDetector(
                onTap: _loadTree,
                child: const Center(child: Column(mainAxisSize: MainAxisSize.min, children: [
                  CircularProgressIndicator(strokeWidth: 2),
                  SizedBox(height: 8),
                  Text('加载中，点击重试', style: TextStyle(fontSize: 11, color: textMuted)),
                ])))
            : _treeData.isEmpty
                ? GestureDetector(
                    onTap: widget.divideId.isNotEmpty ? _loadTree : null,
                    child: Center(
                      child: Column(mainAxisSize: MainAxisSize.min, children: [
                        Icon(Icons.account_tree_outlined, size: 36, color: textMuted.withAlpha(100)),
                        const SizedBox(height: 8),
                        Text(widget.divideId.isEmpty ? '选宿未开放' : '暂无数据，点击重试',
                            style: const TextStyle(fontSize: 12, color: textMuted)),
                      ]),
                    ))
                : ListView(
                    padding: const EdgeInsets.all(8),
                    children: _treeData.map((node) => _buildNode(node as Map<String, dynamic>, depth: 0)).toList(),
                  ),
      ),
    );
  }

  Widget _buildNode(Map<String, dynamic> node, {int depth = 0}) {
    final name = (node['name'] ?? node['label'] ?? node['text'] ?? '?').toString();
    final code = (node['code'] ?? node['value'] ?? node['id'] ?? '').toString();
    final nodeType = (node['type'] ?? '').toString();
    final children = (node['children'] ?? []) as List;

    final isRoom = nodeType == 'ROOM';
    final isLeaf = isRoom || (node['children'] == null &&
        !['ROOT', 'CAMPUS', 'PARK', 'BUILDING', 'UNIT', 'FLOOR'].contains(nodeType));

    if (isLeaf) {
      final isSelected = _selectedRoom == code;
      return Padding(
        padding: EdgeInsets.only(left: 16.0 + depth * 16),
        child: ListTile(
          dense: true,
          contentPadding: EdgeInsets.zero,
          visualDensity: const VisualDensity(horizontal: -4, vertical: -4),
          selected: isSelected,
          selectedTileColor: primaryColor.withAlpha(20),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(6)),
          leading: Icon(Icons.meeting_room_rounded,
              size: 14, color: isSelected ? primaryColor : textMuted),
          title: Text(name,
              style: TextStyle(fontSize: 12,
                  fontWeight: isSelected ? FontWeight.w600 : FontWeight.w400,
                  color: isSelected ? primaryColor : textSecondary)),
          onTap: () {
            setState(() => _selectedRoom = code);
            widget.onRoomSelected(code);
          },
        ),
      );
    }

    IconData icon;
    switch (nodeType) {
      case 'ROOT':
        icon = Icons.school_rounded;
        break;
      case 'CAMPUS':
        icon = Icons.location_city_rounded;
        break;
      case 'PARK':
      case 'BUILDING':
        icon = Icons.apartment_rounded;
        break;
      case 'UNIT':
        icon = Icons.view_day_rounded;
        break;
      case 'FLOOR':
        icon = Icons.layers_rounded;
        break;
      default:
        icon = Icons.folder_rounded;
    }

    return Theme(
      data: Theme.of(context).copyWith(dividerColor: Colors.transparent),
      child: ExpansionTile(
        tilePadding: EdgeInsets.only(left: 8.0 + depth * 16),
        childrenPadding: EdgeInsets.zero,
        visualDensity: const VisualDensity(horizontal: -4, vertical: -4),
        leading: Icon(icon, size: 16, color: depth < 2 ? primaryColor : textSecondary),
        title: Text(name,
            style: TextStyle(
                fontSize: 12 + (depth < 2 ? 1 : 0),
                fontWeight: depth < 2 ? FontWeight.w600 : FontWeight.w500,
                color: depth < 3 ? textPrimary : textSecondary)),
        initiallyExpanded: depth < 3,
        children: children
            .map((c) => _buildNode(c as Map<String, dynamic>, depth: depth + 1))
            .toList(),
      ),
    );
  }
}
