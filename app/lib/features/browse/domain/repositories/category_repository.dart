import 'package:fpdart/fpdart.dart';

import '../../../../core/error/failure.dart';
import '../entities/category.dart';

abstract interface class CategoryRepository {
  Future<Either<Failure, List<Category>>> listCategories({
    bool withCounts,
    int? depth,
    String? rootDomain,
  });
}
