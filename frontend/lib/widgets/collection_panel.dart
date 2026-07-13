import 'package:flutter/material.dart';
import '../theme/app_theme.dart';

class CollectionPanel extends StatelessWidget {
  final List<Map<String, dynamic>> collection;
  final int totalConcurrency;
  final bool readOnly;
  final void Function(List<Map<String, dynamic>>, int) onChanged;

  const CollectionPanel({
    super.key,
    required this.collection,
    required this.totalConcurrency,
    required this.readOnly,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final concurrencyController = TextEditingController(text: '$totalConcurrency');

    return Container(
      constraints: const BoxConstraints(maxHeight: 200),
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
                      return Container(
                        margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                        decoration: BoxDecoration(
                          color: surfaceSecondary,
                          borderRadius: BorderRadius.circular(radiusSm),
                          border: Border.all(color: const Color(0xFFF1F5F9)),
                        ),
                        child: Row(children: [
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
                          Expanded(
                            child: Text(bed['bedName']?.toString() ?? bed['bedCode']?.toString() ?? '?',
                                style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: textPrimary)),
                          ),
                          _buildPriorityDropdown(bed, i),
                          const SizedBox(width: 4),
                          if (!readOnly)
                            InkWell(
                              onTap: () {
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
