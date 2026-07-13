import 'dart:async';
import 'package:flutter/material.dart';
import '../services/api_service.dart';
import '../theme/app_theme.dart';

class GrabPanel extends StatefulWidget {
  final ApiService api;
  final String personsn;
  final String divideId;
  final int totalConcurrency;
  final int collectionCount;
  final Map<String, dynamic>? grabStatus;
  final void Function(Map<String, dynamic>?) onStatusChanged;

  const GrabPanel({
    super.key,
    required this.api,
    required this.personsn,
    required this.divideId,
    required this.totalConcurrency,
    required this.collectionCount,
    required this.grabStatus,
    required this.onStatusChanged,
  });

  @override
  State<GrabPanel> createState() => _GrabPanelState();
}

class _GrabPanelState extends State<GrabPanel> {
  Timer? _pollTimer;

  bool get _isGrabbing => widget.grabStatus != null && widget.grabStatus!['running'] == true;

  @override
  void dispose() {
    _pollTimer?.cancel();
    super.dispose();
  }

  void _startPolling() {
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(const Duration(seconds: 1), (_) async {
      if (!mounted) return;
      try {
        final status = await widget.api.grabStatus();
        if (mounted) {
          widget.onStatusChanged(status);
          if (status['running'] == false) {
            _pollTimer?.cancel();
            _showResult(status);
          }
        }
      } catch (_) {}
    });
  }

  void _showResult(Map<String, dynamic> status) {
    if (!mounted) return;
    final success = status['success'] == true;
    final bed = status['successBed']?.toString() ?? '';
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(success ? '抢床成功!' : '抢床结束'),
        content: Text(success ? '成功抢到: $bed' : '所有床位均未抢到'),
        actions: [
          FilledButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('确定'),
          ),
        ],
      ),
    );
  }

  Future<void> _toggleGrab() async {
    if (_isGrabbing) {
      await widget.api.grabStop();
      _pollTimer?.cancel();
      widget.onStatusChanged(null);
    } else {
      if (widget.collectionCount == 0) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('请先收藏床位'), backgroundColor: warningColor),
          );
        }
        return;
      }
      if (widget.divideId.isEmpty) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('无法获取选宿批次信息'), backgroundColor: dangerColor),
          );
        }
        return;
      }
      await widget.api.grabStart(
        personsn: widget.personsn,
        divideId: widget.divideId,
        totalConcurrency: widget.totalConcurrency,
      );
      _startPolling();
      widget.onStatusChanged({'running': true, 'progress': {}, 'log': []});
    }
  }

  @override
  Widget build(BuildContext context) {
    final status = widget.grabStatus;
    final progressRaw = status?['progress'];
    final Map<String, dynamic> progress = progressRaw is Map
        ? Map<String, dynamic>.from(progressRaw)
        : {};
    final List<String> logs = (status?['log'] is List)
        ? List<String>.from(status!['log'])
        : [];

    return Container(
      decoration: const BoxDecoration(
        color: surfaceColor,
        border: Border(top: BorderSide(color: borderColor)),
      ),
      padding: const EdgeInsets.all(12),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Row(children: [
            const Text('抢床控制', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: textPrimary)),
            const Spacer(),
            SizedBox(
              height: 36,
              child: DecoratedBox(
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(radiusMd),
                  gradient: _isGrabbing
                      ? const LinearGradient(colors: [dangerColor, Color(0xFFDC2626)])
                      : primaryGradient,
                ),
                child: FilledButton(
                  onPressed: _toggleGrab,
                  style: FilledButton.styleFrom(
                    backgroundColor: Colors.transparent,
                    shadowColor: Colors.transparent,
                    padding: const EdgeInsets.symmetric(horizontal: 16),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(_isGrabbing ? Icons.stop_rounded : Icons.play_arrow_rounded, size: 20),
                      const SizedBox(width: 4),
                      Text(_isGrabbing ? '停 止' : '开 始',
                          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600)),
                    ],
                  ),
                ),
              ),
            ),
          ]),
          if (progress.isNotEmpty) ...[
            const SizedBox(height: 8),
            ...progress.entries.map((e) {
              final bp = (e.value is Map) ? Map<String, dynamic>.from(e.value) : <String, dynamic>{};
              final done = (bp['done'] as num?)?.toInt() ?? 0;
              final total = (bp['total'] as num?)?.toInt() ?? 0;
              final ok = (bp['ok'] as num?)?.toInt() ?? 0;
              final fail = (bp['fail'] as num?)?.toInt() ?? 0;
              final ratio = total > 0 ? done / total : 0.0;
              return Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: Row(children: [
                  Flexible(
                    flex: 2,
                    child: Text(e.key, style: const TextStyle(fontSize: 11, fontFamily: 'monospace', color: textSecondary), overflow: TextOverflow.ellipsis),
                  ),
                  const SizedBox(width: 4),
                  Flexible(
                    flex: 5,
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(2),
                      child: LinearProgressIndicator(
                        value: ratio, minHeight: 6,
                        backgroundColor: borderColor,
                        valueColor: const AlwaysStoppedAnimation<Color>(primaryColor),
                      ),
                    ),
                  ),
                  const SizedBox(width: 4),
                  Text('$ok/$fail', style: const TextStyle(fontSize: 10, color: textMuted)),
                ]),
              );
            }),
          ],
          if (logs.isNotEmpty) ...[
            const SizedBox(height: 8),
            SizedBox(
              height: 80,
              child: ListView.builder(
                itemCount: logs.length,
                itemBuilder: (_, i) => Text(logs[i],
                    style: const TextStyle(fontSize: 11, color: textSecondary, fontFamily: 'monospace')),
              ),
            ),
          ],
        ],
      ),
    );
  }
}
