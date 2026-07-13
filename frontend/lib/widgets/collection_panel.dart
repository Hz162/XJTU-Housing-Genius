import 'package:flutter/material.dart';
import '../services/api_service.dart';
import '../theme/app_theme.dart';

class CollectionPanel extends StatelessWidget {
  final ApiService api;
  final List<Map<String, dynamic>> collection;
  final int totalConcurrency;
  final bool readOnly;
  final void Function(List<Map<String, dynamic>>, int) onChanged;

  const CollectionPanel({
    super.key,
    required this.api,
    required this.collection,
    required this.totalConcurrency,
    required this.readOnly,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final concurrencyController = TextEditingController(text: '$totalConcurrency');

    return Container(
      constraints: const BoxConstraints(maxHeight: 220),
      decoration: const BoxDecoration(
        color: surfaceColor,
        border: Border(top: BorderSide(color: borderColor)),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Row(children: [
              const Text('收藏列表', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: textPrimary)),
              const Spacer(),
              SizedBox(
                width: 60,
                child: TextField(
                  decoration: const InputDecoration(
                    labelText: '并发',
                    isDense: true,
                    contentPadding: EdgeInsets.symmetric(horizontal: 8, vertical: 8),
                  ),
                  keyboardType: TextInputType.number,
                  controller: concurrencyController,
                  readOnly: readOnly,
                  onSubmitted: (v) {
                    final n = int.tryParse(v);
                    if (n != null && n >= collection.length) {
                      onChanged(collection, n);
                    }
                  },
                ),
              ),
            ]),
          ),
          Expanded(
            child: collection.isEmpty
                ? const Center(child: Text('暂无收藏', style: TextStyle(fontSize: 12, color: textMuted)))
                : ListView.builder(
                    itemCount: collection.length,
                    itemBuilder: (_, i) {
                      final bed = collection[i];
                      final num = bed['num']?.toString() ?? '0';
                      final status = bed['status']?.toString() ?? '0';
                      final isTaken = status == '1';

                      return Container(
                        margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                        decoration: BoxDecoration(
                          color: isTaken ? dangerColor.withAlpha(12) : surfaceSecondary,
                          borderRadius: BorderRadius.circular(radiusSm),
                          border: Border.all(color: isTaken ? dangerColor.withAlpha(40) : const Color(0xFFF1F5F9)),
                        ),
                        child: Row(children: [
                          // 序号
                          Container(
                            width: 24, height: 24,
                            decoration: BoxDecoration(
                              color: primaryColor.withAlpha(20),
                              borderRadius: BorderRadius.circular(6),
                            ),
                            child: Center(
                              child: Text('${i + 1}',
                                  style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w700, color: primaryColor)),
                            ),
                          ),
                          const SizedBox(width: 8),
                          // 床位名称
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(bed['bedName']?.toString() ?? bed['bedCode']?.toString() ?? '?',
                                    style: TextStyle(
                                      fontSize: 12.5,
                                      fontWeight: FontWeight.w500,
                                      color: isTaken ? textMuted : textPrimary,
                                      decoration: isTaken ? TextDecoration.lineThrough : null,
                                    ),
                                    maxLines: 1, overflow: TextOverflow.ellipsis),
                                const SizedBox(height: 2),
                                Row(children: [
                                  // 收藏人数 (原网页 num)
                                  Icon(Icons.people_outline_rounded, size: 10, color: textMuted),
                                  const SizedBox(width: 2),
                                  Text('$num人收藏',
                                      style: const TextStyle(fontSize: 10, color: textSecondary)),
                                  const SizedBox(width: 8),
                                  // 状态标签
                                  Container(
                                    padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
                                    decoration: BoxDecoration(
                                      color: isTaken ? dangerColor.withAlpha(20) : successColor.withAlpha(20),
                                      borderRadius: BorderRadius.circular(3),
                                    ),
                                    child: Text(
                                      isTaken ? '已被抢' : '可用',
                                      style: TextStyle(
                                        fontSize: 9,
                                        fontWeight: FontWeight.w600,
                                        color: isTaken ? dangerColor : successColor,
                                      ),
                                    ),
                                  ),
                                ]),
                              ],
                            ),
                          ),
                          // 优先级下拉
                          _buildPriorityDropdown(bed, i),
                          const SizedBox(width: 4),
                          // 删除按钮
                          if (!readOnly)
                            InkWell(
                              onTap: () async {
                                // 从服务器删除
                                final serverId = bed['serverId']?.toString() ?? '';
                                final bedCode = bed['bedCode']?.toString() ?? '';
                                if (serverId.isNotEmpty) {
                                  try { api.collectSyncDelete(id: serverId, bedCode: bedCode); } catch (_) {}
                                }
                                final newCol = List<Map<String, dynamic>>.from(collection);
                                newCol.removeAt(i);
                                onChanged(newCol, totalConcurrency);
                              },
                              child: const Icon(Icons.delete_outline_rounded, size: 18, color: dangerColor),
                            ),
                        ]),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }

  Widget _buildPriorityDropdown(Map<String, dynamic> bed, int index) {
    final current = (bed['priority'] as int?) ?? 1;
    return PopupMenuButton<int>(
      initialValue: current,
      enabled: !readOnly,
      onSelected: (v) {
        final newCol = List<Map<String, dynamic>>.from(collection);
        newCol[index]['priority'] = v;
        onChanged(newCol, totalConcurrency);
      },
      itemBuilder: (_) => List.generate(5, (i) => i + 1).map((p) =>
        PopupMenuItem(value: p, height: 32,
          child: Text('优先级 $p', style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: textPrimary)),
        ),
      ).toList(),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: surfaceSecondary,
          borderRadius: BorderRadius.circular(radiusSm),
          border: Border.all(color: borderColor),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('$current', style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: textPrimary)),
            const SizedBox(width: 2),
            const Icon(Icons.arrow_drop_down_rounded, size: 15, color: textSecondary),
          ],
        ),
      ),
    );
  }
}
