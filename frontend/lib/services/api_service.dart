import 'dart:convert';
import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

/// Thrown when the backend session has expired and auto-re-login failed.
class SessionExpiredException implements Exception {
  @override
  String toString() => '会话已过期，请重新登录';
}

class ApiService {
  static const _defaultPort = 18721;

  String _baseUrl = 'http://127.0.0.1:$_defaultPort/api';

  String get baseUrl => _baseUrl;

  final http.Client _client = http.Client();

  // ── Port discovery ──

  static String get _configDir {
    if (Platform.isWindows) {
      final appdata = Platform.environment['APPDATA'] ??
          '${Platform.environment['USERPROFILE']}\\AppData\\Roaming';
      return '$appdata\\xjtu-housing-genius';
    } else if (Platform.isMacOS) {
      return '${Platform.environment['HOME']}/Library/Application Support/xjtu-housing-genius';
    } else {
      final xdg = Platform.environment['XDG_CONFIG_HOME'] ??
          '${Platform.environment['HOME']}/.config';
      return '$xdg/xjtu-housing-genius';
    }
  }

  /// Connect directly to a backend at the given port.
  void connectPort(int port) {
    _baseUrl = 'http://127.0.0.1:$port/api';
  }

  /// Auto-discover backend: first try known port, then scan.
  Future<String> discover() async {
    // 1) Try reading port file written by backend
    try {
      final portFile =
          File('$_configDir${Platform.pathSeparator}port');
      if (await portFile.exists()) {
        final port = (await portFile.readAsString()).trim();
        if (port.isNotEmpty) {
          final url = 'http://127.0.0.1:$port/api';
          if (await _ping(url)) {
            _baseUrl = url;
            return _baseUrl;
          }
        }
      }
    } catch (_) {}

    // 2) Try default port and nearby ports
    for (var p = _defaultPort; p < _defaultPort + 30; p++) {
      final url = 'http://127.0.0.1:$p/api';
      if (await _ping(url)) {
        _baseUrl = url;
        return _baseUrl;
      }
    }

    return _baseUrl;
  }

  Future<bool> _ping(String url) async {
    try {
      final resp = await http
          .get(Uri.parse('$url/session/check'))
          .timeout(const Duration(milliseconds: 800));
      return resp.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  // ── HTTP helpers ──

  bool _reloginInProgress = false;

  Future<dynamic> _request(
    String method,
    String path, {
    Map<String, dynamic>? body,
  }) async {
    final uri = Uri.parse('$_baseUrl$path');
    http.Response resp;
    final headers = {'Content-Type': 'application/json'};

    switch (method) {
      case 'GET':
        resp = await _client.get(uri, headers: headers);
        break;
      case 'POST':
        resp = await _client.post(uri,
            headers: headers, body: jsonEncode(body));
        break;
      default:
        throw Exception('Unknown method $method');
    }

    // Auto re-login on session expiry
    if (resp.body.trimLeft().startsWith('<') && !_reloginInProgress) {
      _reloginInProgress = true;
      try {
        debugPrint('[ApiService] session expired, auto relogin...');
        final reloginOk = await _tryRelogin();
        if (reloginOk) {
          debugPrint('[ApiService] relogin OK, retrying $path');
          return _request(method, path, body: body);
        }
      } finally {
        _reloginInProgress = false;
      }
      throw SessionExpiredException();
    }

    if (resp.statusCode >= 400) {
      try {
        final err = jsonDecode(resp.body);
        throw Exception(err['error'] ?? '请求失败');
      } catch (e) {
        if (e is SessionExpiredException ||
            e.toString().contains('会话')) {
        rethrow;
      }
        throw Exception('服务器错误 (${resp.statusCode})');
      }
    }

    try {
      return jsonDecode(resp.body);
    } catch (_) {
      throw Exception('服务器返回异常数据');
    }
  }

  Future<bool> _tryRelogin() async {
    try {
      final resp = await _client.post(
        Uri.parse('$_baseUrl/relogin'),
        headers: {'Content-Type': 'application/json'},
      );
      if (resp.statusCode == 200) {
        final data = jsonDecode(resp.body);
        return data['status'] == 'ok';
      }
    } catch (_) {}
    return false;
  }

  // ── Login & MFA ──

  Future<Map<String, dynamic>> login(String account, String password,
      {String captcha = ''}) async {
    final data = await _request('POST', '/login',
        body: {
          'account': account,
          'password': password,
          'captcha': captcha
        });
    return data as Map<String, dynamic>;
  }

  Future<List<int>> getCaptchaImage() async {
    final uri = Uri.parse('$_baseUrl/captcha');
    final resp = await _client.get(uri);
    if (resp.statusCode == 200) {
      return resp.bodyBytes.toList();
    }
    throw Exception('获取验证码失败');
  }

  Future<Map<String, dynamic>> mfaInit(String method) async {
    final data = await _request('POST', '/mfa/init', body: {'method': method});
    return data as Map<String, dynamic>;
  }

  Future<void> mfaSend() => _request('POST', '/mfa/send');

  Future<Map<String, dynamic>> mfaVerify(String code) async {
    final data = await _request('POST', '/mfa/verify', body: {'code': code});
    return data as Map<String, dynamic>;
  }

  Future<Map<String, dynamic>> chooseAccount(String accountType) async {
    final data = await _request('POST', '/account/choose',
        body: {'accountType': accountType});
    return data as Map<String, dynamic>;
  }

  // ── Session ──

  Future<bool> checkSession() async {
    final data = await _request('GET', '/session/check');
    return data['alive'] ?? false;
  }

  Future<void> relogin() => _request('POST', '/relogin');

  // ── Config ──

  Future<Map<String, dynamic>> getConfig() async {
    final data = await _request('GET', '/config');
    return data as Map<String, dynamic>;
  }

  // ── Bed ──

  Future<Map<String, dynamic>> getDivideId(String personsn) async {
    final data = await _request('GET', '/bed/divideId?personsn=$personsn');
    return data as Map<String, dynamic>;
  }

  Future<List<dynamic>> getBedTree(String divideId) async {
    final data = await _request('GET', '/bed/tree?divideId=$divideId');
    return (data is List) ? data : <dynamic>[];
  }

  Future<Map<String, dynamic>> getRoomBeds(String divideId, String roomCode) async {
    final data = await _request(
        'GET', '/bed/room-beds?divideId=$divideId&roomCode=$roomCode');
    return data as Map<String, dynamic>;
  }

  Future<Map<String, dynamic>> checkMyBed(String personsn, String divideId) async {
    final data = await _request(
        'GET', '/bed/check?personsn=$personsn&divideId=$divideId');
    return data as Map<String, dynamic>;
  }

  Future<Map<String, dynamic>> getCollection() async {
    final data = await _request('GET', '/bed/collection');
    return data as Map<String, dynamic>;
  }

  Future<void> saveCollection(Map<String, dynamic> collection) =>
      _request('POST', '/bed/collection', body: collection);

  Future<Map<String, dynamic>> grabStart(
      {required String personsn, required String divideId, required int totalConcurrency}) async {
    final data = await _request('POST',
        '/bed/grab/start?personsn=$personsn&divideId=$divideId',
        body: {'totalConcurrency': totalConcurrency});
    return data as Map<String, dynamic>;
  }

  Future<void> grabStop() => _request('POST', '/bed/grab/stop');

  Future<Map<String, dynamic>> grabStatus() async {
    final data = await _request('GET', '/bed/grab/status');
    return data as Map<String, dynamic>;
  }

  Future<Map<String, dynamic>> getRoomAssign(String divideId, String roomCodes) async {
    final data = await _request(
        'GET', '/bed/room-assign?divideId=$divideId&roomCodes=$roomCodes');
    return data as Map<String, dynamic>;
  }
}
