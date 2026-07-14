import 'package:dio/dio.dart';

import '../../../../core/network/api_envelope.dart';
import '../dtos/category_dto.dart';

class CategoryRemoteDataSource {
  const CategoryRemoteDataSource(this._dio);

  final Dio _dio;

  /// `GET /categories`. [withCounts] populates `productCount` (only for root
  /// domains, depth 0); [depth] filters by tree level (`0` = root domains);
  /// [rootDomain] scopes to one domain. Null params are omitted from the query.
  Future<List<CategoryDto>> getCategories({
    bool withCounts = false,
    int? depth,
    String? rootDomain,
    String? parentId,
  }) async {
    final query = <String, dynamic>{};
    if (withCounts) query['withCounts'] = true;
    if (depth != null) query['depth'] = depth;
    if (rootDomain != null) query['rootDomain'] = rootDomain;
    if (parentId != null && parentId.isNotEmpty) query['parentId'] = parentId;
    final response = await _dio.get<dynamic>(
      '/categories',
      queryParameters: query.isEmpty ? null : query,
    );
    final data = unwrap(response) as List<dynamic>;
    return data
        .map((e) => CategoryDto.fromJson(e as Map<String, dynamic>))
        .toList();
  }
}
