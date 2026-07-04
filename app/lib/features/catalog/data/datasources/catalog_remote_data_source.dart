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
}
