import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/providers.dart';
import '../data/datasources/category_remote_data_source.dart';
import '../data/repositories/category_repository_impl.dart';
import '../data/repositories/notification_repository_stub.dart';
import '../domain/entities/app_notification.dart';
import '../domain/entities/category.dart';
import '../domain/repositories/category_repository.dart';
import '../domain/repositories/notification_repository.dart';
import '../domain/usecases/get_categories.dart';

// ---- Categories (real) ----

final categoryRemoteDataSourceProvider = Provider<CategoryRemoteDataSource>(
  (ref) => CategoryRemoteDataSource(ref.watch(dioProvider)),
);

final categoryRepositoryProvider = Provider<CategoryRepository>(
  (ref) => CategoryRepositoryImpl(ref.watch(categoryRemoteDataSourceProvider)),
);

final getCategoriesUseCaseProvider = Provider<GetCategories>(
  (ref) => GetCategories(ref.watch(categoryRepositoryProvider)),
);

/// Root-domain categories with product counts (`GET /categories?depth=0&
/// withCounts=true`). Counts are only populated for root domains, so the
/// categories grid requests depth 0. Throws the [Failure] so the UI renders it
/// via the AsyncValue error state (mirrors `walletProvider`).
final categoriesProvider = FutureProvider.autoDispose<List<Category>>((ref) async {
  final result = await ref
      .watch(getCategoriesUseCaseProvider)
      .call(depth: 0, withCounts: true);
  return result.match((failure) => throw failure, (categories) => categories);
});

// ---- Notifications (stub — no backend endpoint yet) ----

final notificationRepositoryProvider = Provider<NotificationRepository>(
  // TODO(backend): swap [NotificationRepositoryStub] for an HTTP impl when a
  // GET /notifications endpoint exists — this is the single wiring point.
  (ref) => const NotificationRepositoryStub(),
);

/// In-app notifications. Today the stub returns a small static sample so the
/// populated state renders; the empty state is reachable when the list is empty.
final notificationsProvider =
    FutureProvider.autoDispose<List<AppNotification>>((ref) async {
  final result = await ref.watch(notificationRepositoryProvider).listNotifications();
  return result.match((failure) => throw failure, (items) => items);
});
