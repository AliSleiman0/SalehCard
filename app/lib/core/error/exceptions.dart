/// Raised when the API returns a well-formed error envelope
/// (`{success:false, error:{code,message}}`) or an unexpected body shape.
class ApiException implements Exception {
  const ApiException({
    required this.code,
    required this.message,
    this.statusCode,
  });

  final String code;
  final String message;
  final int? statusCode;

  @override
  String toString() => 'ApiException($code, $statusCode): $message';
}
