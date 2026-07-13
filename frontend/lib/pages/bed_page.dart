import 'dart:async';
import 'package:flutter/material.dart';
import '../main.dart';
import '../services/api_service.dart';
import '../theme/app_theme.dart';
import '../widgets/window_bar.dart';
import '../widgets/bed_sidebar.dart';
import '../widgets/bed_content.dart';
import '../widgets/collection_panel.dart';
import '../widgets/grab_panel.dart';

class BedPage extends StatefulWidget {
  final ApiService api;
  final String studentCode;
  const BedPage({super.key, required this.api, required this.studentCode});

  @override
  State<BedPage> createState() => _BedPageState();
}

class _BedPageState extends State<BedPage> {
  late ApiService api;

  bool _sidebarCollapsed = false;
  String _divideId = '';
  String _personsn = '';
  String? _selectedRoomCode;
  bool _isMyBed = false;
  bool _initialized = false;

  List<Map<String, dynamic>> _collection = [];
  int _totalConcurrency = 10;
  Map<String, dynamic>? _grabStatus;
  Timer? _grabTimer;

  @override
  void initState() {
    super.initState();
    api = widget.api;
    _initData();
    _startSessionMonitor();
  }

  @override
  void dispose() {
    _grabTimer?.cancel();
    _sessionTimer?.cancel();
    super.dispose();
  }

  Future<void> _initData() async {
    try {
      _personsn = widget.studentCode;

      final divResp = await api.getDivideId(_personsn);
      if (divResp['code'] == 0 && divResp['map'] != null) {
        _divideId = divResp['map']['divideId']?.toString() ?? '';
      } else if (divResp['divideId'] != null) {
        _divideId = divResp['divideId'].toString();
      }

      if (_divideId.isNotEmpty) {
        final checkResp = await api.checkMyBed(_personsn, _divideId);
        _isMyBed = checkResp['isMybed'] == true || checkResp['data']?['isMybed'] == true;
      }

      final colResp = await api.getCollection();
      if (colResp['beds'] != null) {
        _collection = List<Map<String, dynamic>>.from(colResp['beds']);
      }
      _totalConcurrency = (colResp['totalConcurrency'] as int?) ?? 10;

      setState(() => _initialized = true);
    } catch (e) {
      if (mounted) {
        setState(() => _initialized = true);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('初始化失败: $e'), backgroundColor: dangerColor),
        );
      }
    }
  }

  Timer? _sessionTimer;
  bool _sessionDead = false;

  void _startSessionMonitor() {
    _sessionTimer = Timer.periodic(const Duration(seconds: 30), (_) async {
      try {
        final alive = await api.checkSession();
        if (!alive && mounted && !_sessionDead) {
          setState(() => _sessionDead = true);
          final ok = await api.relogin().then((_) => true).catchError((_) => false);
          if (ok) {
            setState(() => _sessionDead = false);
          } else {
            if (mounted) redirectToLogin();
          }
        }
      } catch (_) {}
    });
  }

  void _onRoomSelected(String roomCode) {
    setState(() => _selectedRoomCode = roomCode);
  }

  void _onCollectionChanged(List<Map<String, dynamic>> collection, int concurrency) {
    setState(() {
      _collection = collection;
      _totalConcurrency = concurrency;
    });
  }

  void _onGrabStatus(Map<String, dynamic>? status) {
    setState(() => _grabStatus = status);
  }

  bool get _isGrabbing => _grabStatus != null && _grabStatus!['running'] == true;

  @override
  Widget build(BuildContext context) {
    if (!_initialized) {
      return const Scaffold(
        backgroundColor: bgColor,
        body: Center(child: CircularProgressIndicator()),
      );
    }

    return Scaffold(
      backgroundColor: bgColor,
      body: Column(
        children: [
          WindowBar(
            leading: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                IconButton(
                  icon: Icon(
                    _sidebarCollapsed ? Icons.menu_rounded : Icons.menu_open_rounded,
                    size: 20,
                  ),
                  onPressed: () => setState(() => _sidebarCollapsed = !_sidebarCollapsed),
                ),
                const Text('XJTU Housing Genius',
                    style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: textPrimary)),
                if (_isGrabbing)
                  Padding(
                    padding: const EdgeInsets.only(left: 8),
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                      decoration: BoxDecoration(
                        gradient: primaryGradient,
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: const Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          SizedBox(width: 10, height: 10,
                              child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white)),
                          SizedBox(width: 6),
                          Text('抢床中', style: TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.w600)),
                        ],
                      ),
                    ),
                  ),
              ],
            ),
          ),
          Expanded(
            child: Row(
              children: [
                BedSidebar(
                  collapsed: _sidebarCollapsed,
                  api: api,
                  divideId: _divideId,
                  onRoomSelected: _onRoomSelected,
                ),
                Expanded(
                  child: Column(
                    children: [
                      Expanded(
                        child: _isMyBed
                            ? Center(
                                child: Column(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    Icon(Icons.bed_rounded, size: 64, color: successColor.withAlpha(150)),
                                    const SizedBox(height: 16),
                                    const Text('您已有床位', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600, color: textPrimary)),
                                    const SizedBox(height: 8),
                                    const Text('选床系统已为您分配床位', style: TextStyle(fontSize: 14, color: textSecondary)),
                                  ],
                                ),
                              )
                            : _selectedRoomCode != null
                                ? BedContent(
                                    api: api,
                                    divideId: _divideId,
                                    roomCode: _selectedRoomCode!,
                                    personsn: _personsn,
                                    collection: _collection,
                                    onCollectionChanged: _onCollectionChanged,
                                    readOnly: _isGrabbing,
                                  )
                                : Center(
                                    child: Column(
                                      mainAxisSize: MainAxisSize.min,
                                      children: [
                                        Icon(Icons.arrow_back_rounded, size: 48, color: textMuted.withAlpha(100)),
                                        const SizedBox(height: 12),
                                        const Text('从左侧选择一个房间', style: TextStyle(fontSize: 14, color: textMuted)),
                                      ],
                                    ),
                                  ),
                      ),
                      CollectionPanel(
                        collection: _collection,
                        totalConcurrency: _totalConcurrency,
                        readOnly: _isGrabbing,
                        onChanged: (col, concurrency) {
                          _onCollectionChanged(col, concurrency);
                          api.saveCollection({
                            'beds': col,
                            'totalConcurrency': concurrency,
                          });
                        },
                      ),
                      GrabPanel(
                        api: api,
                        personsn: _personsn,
                        divideId: _divideId,
                        totalConcurrency: _totalConcurrency,
                        grabStatus: _grabStatus,
                        collectionCount: _collection.length,
                        onStatusChanged: _onGrabStatus,
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
