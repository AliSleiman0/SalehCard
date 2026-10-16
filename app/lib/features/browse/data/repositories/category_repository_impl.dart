import 'package:fpdart/fpdart.dart';

import '../../../../core/config/regulated_finance.dart';
import '../../../../core/error/failure.dart';
import '../../domain/entities/category.dart';
import '../../domain/repositories/category_repository.dart';
import '../datasources/category_remote_data_source.dart';

class CategoryRepositoryImpl implements CategoryRepository {
  const CategoryRepositoryImpl(this._remote);

  final CategoryRemoteDataSource _remote;

  @override
  Future<Either<Failure, List<Category>>> listCategories({
    bool withCounts = false,
    int? depth,
    String? rootDomain,
    String? parentId,
  }) async {
    try {
      final dtos = await _remote.getCategories(
        withCounts: withCounts,
        depth: depth,
        rootDomain: rootDomain,
        parentId: parentId,
      );
      return Right([
        for (final category in dtos.map((d) => d.toEntity()))
          if (!hideRegulatedFinance ||
              !regulatedRootDomains.contains(category.rootDomain))
            category,
      ]);
    } catch (error) {
      return Left(mapError(error));
    }
  }
}
