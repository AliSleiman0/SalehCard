/// Static application configuration.
///
/// The API base URL is overridable at build/run time via
/// `--dart-define=API_BASE_URL=...`. The default targets the dev API over the
/// LAN so a physical Android device on the same network can reach it (a device
/// cannot use `localhost`/`10.0.2.2`). Update the default if the dev machine's
/// LAN IP changes.
class AppConfig {
  const AppConfig._();

  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://192.168.10.170:8090/api/v1',
  );

  /// Header value identifying this native client to the backend so it returns
  /// the refresh token in the JSON body (browsers use the httpOnly cookie).
  static const String clientHeaderName = 'X-Client';
  static const String clientHeaderValue = 'mobile';
}
