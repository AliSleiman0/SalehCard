import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/category.dart';
import '../repositories/category_repository.dart';

class GetCategories {
  const GetCategories(this._repository);

  final CategoryRepository _repository;

  Future<Either<Failure, List<Category>>> call({
    bool withCounts = false,
    int? depth,
    String? rootDomain,
    String? parentId,
  }) {
    return _repository.listCategories(
      withCounts: withCounts,
      depth: depth,
      rootDomain: rootDomain,
      parentId: parentId,
    );
  }
}
