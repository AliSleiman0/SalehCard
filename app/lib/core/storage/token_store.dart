import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Persists the JWT access token and the opaque refresh token in the platform
/// secure store. The refresh token is in the body (not an httpOnly cookie)
/// because native clients identify themselves with `X-Client: mobile`.
class TokenStore {
  const TokenStore(this._storage);

  final FlutterSecureStorage _storage;

  static const _accessKey = 'access_token';
  static const _refreshKey = 'refresh_token';

  Future<String?> readAccess() => _storage.read(key: _accessKey);

  Future<String?> readRefresh() => _storage.read(key: _refreshKey);

  Future<void> save({required String access, String? refresh}) async {
    await _storage.write(key: _accessKey, value: access);
    if (refresh != null && refresh.isNotEmpty) {
      await _storage.write(key: _refreshKey, value: refresh);
    }
  }

  Future<void> clear() async {
    await _storage.delete(key: _accessKey);
    await _storage.delete(key: _refreshKey);
  }
}
