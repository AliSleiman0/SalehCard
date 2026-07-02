import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/network/providers.dart';
import '../data/datasources/category_remote_data_source.dart';
import '../data/datasources/notification_remote_data_source.dart';
import '../data/repositories/category_repository_impl.dart';
import '../data/repositories/notification_repository_impl.dart';
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
final categoriesProvider = FutureProvider.autoDispose<List<Category>>((
  ref,
) async {
  final result = await ref
      .watch(getCategoriesUseCaseProvider)
      .call(depth: 0, withCounts: true);
  return result.match((failure) => throw failure, (categories) => categories);
});

// ---- Notifications (real — GET /notifications) ----

final notificationRepositoryProvider = Provider<NotificationRepository>(
  (ref) => NotificationRepositoryImpl(
    NotificationRemoteDataSource(ref.watch(dioProvider)),
  ),
);

/// The caller's in-app notification inbox, newest first.
final notificationsProvider = FutureProvider.autoDispose<List<AppNotification>>(
  (ref) async {
    final result = await ref
        .watch(notificationRepositoryProvider)
        .listNotifications();
    return result.match((failure) => throw failure, (items) => items);
  },
);

/// Unread-notification count for the bell badge. Failures render as 0 — the
/// badge must never surface an error state.
final unreadNotificationsCountProvider = FutureProvider.autoDispose<int>((
  ref,
) async {
  final result = await ref.watch(notificationRepositoryProvider).unreadCount();
  return result.match((_) => 0, (count) => count);
});
