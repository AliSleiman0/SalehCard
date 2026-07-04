import 'package:dio/dio.dart';

import '../../../../core/network/api_envelope.dart';
import '../dtos/product_dto.dart';

class CatalogRemoteDataSource {
  const CatalogRemoteDataSource(this._dio);

  final Dio _dio;

  Future<List<ProductDto>> getProducts({
    int page = 1,
    int limit = 20,
    String? category,
    String? rootDomain,
  }) async {
    final response = await _dio.get<dynamic>(
      '/products',
      queryParameters: {
        'page': page,
        'limit': limit,
        if (category != null && category.isNotEmpty) 'category': category,
        if (rootDomain != null && rootDomain.isNotEmpty) 'rootDomain': rootDomain,
      },
    );
    final data = unwrap(response) as List<dynamic>;
    return data
        .map((e) => ProductDto.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<ProductDto> getProduct(String id) async {
    final response = await _dio.get<dynamic>('/products/$id');
    return ProductDto.fromJson(unwrap(response) as Map<String, dynamic>);
  }

  /// Resolves a game player ID to its account nickname via the product's
  /// configured verification provider. The backend always returns 200 for the
  /// business outcomes below; only genuine transport failures throw.
  Future<VerifyAccountResult> verifyAccount(
    String productId,
    String playerId,
  ) async {
    final response = await _dio.post<dynamic>(
      '/products/$productId/verify-account',
      data: {'playerId': playerId},
    );
    final data = unwrap(response) as Map<String, dynamic>;
    return VerifyAccountResult(
      found: data['found'] == true,
      username: (data['username'] as String?) ?? '',
      banned: data['banned'] == true,
      reason: (data['reason'] as String?) ?? '',
    );
  }
}

/// Outcome of a verify-account call. [found] is false both for a positively-
/// unknown id ([reason] == "id_not_found") and when the upstream check is
/// unavailable ([reason] == "unavailable").
class VerifyAccountResult {
  const VerifyAccountResult({
    required this.found,
    this.username = '',
    this.banned = false,
    this.reason = '',
  });

  final bool found;
  final String username;
  final bool banned;
  final String reason;
}
