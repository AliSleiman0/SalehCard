import 'package:dio/dio.dart';

import 'exceptions.dart';

/// Domain-level error type returned (via [Either]) from repositories. The
/// presentation layer switches on the concrete subtype to react appropriately.
sealed class Failure {
  const Failure(this.message);
  final String message;
}

class NetworkFailure extends Failure {
  const NetworkFailure(super.message);
}

class UnauthorizedFailure extends Failure {
  const UnauthorizedFailure(super.message);
}

class OutOfStockFailure extends Failure {
  const OutOfStockFailure(super.message);
}

class InsufficientFundsFailure extends Failure {
  const InsufficientFundsFailure(super.message);
}

class ServerFailure extends Failure {
  const ServerFailure(this.code, String message, [this.statusCode])
      : super(message);
  final String code;
  final int? statusCode;
}

class UnknownFailure extends Failure {
  const UnknownFailure(super.message);
}

/// Maps low-level errors (Dio transport errors, API error envelopes) to a
/// [Failure]. Centralised so every repository handles errors identically.
Failure mapError(Object error) {
  if (error is ApiException) {
    return _fromCode(error.code, error.message, error.statusCode);
  }
  if (error is DioException) {
    final data = error.response?.data;
    if (data is Map && data['error'] is Map) {
      final err = data['error'] as Map;
      return _fromCode(
        err['code']?.toString() ?? 'UNKNOWN',
        err['message']?.toString() ?? 'Request failed',
        error.response?.statusCode,
      );
    }
    switch (error.type) {
      case DioExceptionType.connectionError:
      case DioExceptionType.connectionTimeout:
      case DioExceptionType.receiveTimeout:
      case DioExceptionType.sendTimeout:
        return const NetworkFailure(
          'Network error. Check your connection and try again.',
        );
      default:
        return ServerFailure(
          'UNKNOWN',
          error.message ?? 'Server error',
          error.response?.statusCode,
        );
    }
  }
  return UnknownFailure(error.toString());
}

Failure _fromCode(String code, String message, int? status) {
  switch (code) {
    case 'UNAUTHORIZED':
      return UnauthorizedFailure(message);
    case 'OUT_OF_STOCK':
      return OutOfStockFailure(message);
    case 'INSUFFICIENT_FUNDS':
      return InsufficientFundsFailure(message);
    default:
      return ServerFailure(code, message, status);
  }
}
