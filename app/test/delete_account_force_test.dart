import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/error/failure.dart';
import 'package:salehcard_app/features/account/data/datasources/profile_remote_data_source.dart';
import 'package:salehcard_app/features/account/data/repositories/profile_repository_impl.dart';
import 'package:salehcard_app/features/account/presentation/providers.dart';

/// A Dio whose requests never leave the process: every request is recorded in
/// [requests] and answered by [respond] (null = 204 success, otherwise a 409
/// API error envelope carrying that code).
class _FakeApi {
  _FakeApi({this.errorCode});

  String? errorCode;
  final requests = <RequestOptions>[];

  late final Dio dio = Dio(BaseOptions(baseUrl: 'http://test'))
    ..interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requests.add(options);
          final code = errorCode;
          if (code == null) {
            handler.resolve(Response(requestOptions: options, statusCode: 204));
            return;
          }
          handler.reject(
            DioException(
              requestOptions: options,
              type: DioExceptionType.badResponse,
              response: Response(
                requestOptions: options,
                statusCode: 409,
                data: {
                  'error': {'code': code, 'message': code},
                },
              ),
            ),
          );
        },
      ),
    );
}

void main() {
  group('ProfileRemoteDataSource.deleteMe', () {
    test('sends no body without force', () async {
      final api = _FakeApi();
      await ProfileRemoteDataSource(api.dio).deleteMe();
      expect(api.requests.single.method, 'DELETE');
      expect(api.requests.single.path, '/users/me');
      expect(api.requests.single.data, isNull);
    });

    test('sends {"force": true} only when forced', () async {
      final api = _FakeApi();
      await ProfileRemoteDataSource(api.dio).deleteMe(force: true);
      expect(api.requests.single.data, {'force': true});
    });
  });

  group('ProfileRepositoryImpl.deleteAccount', () {
    test('passes force through and maps forceable errors', () async {
      final api = _FakeApi(errorCode: 'WALLET_NOT_EMPTY');
      final repo = ProfileRepositoryImpl(ProfileRemoteDataSource(api.dio));

      final refused = await repo.deleteAccount();
      expect(api.requests.last.data, isNull);
      expect(refused.getLeft().toNullable(), isA<WalletNotEmptyFailure>());

      api.errorCode = null;
      final forced = await repo.deleteAccount(force: true);
      expect(api.requests.last.data, {'force': true});
      expect(forced.isRight(), isTrue);
    });
  });

  group('canForceDelete', () {
    test('offers "Delete anyway" only for wallet balance / orders in flight',
        () {
      expect(canForceDelete(const WalletNotEmptyFailure('x')), isTrue);
      expect(canForceDelete(const OrdersInFlightFailure('x')), isTrue);
      expect(canForceDelete(const PaymentsPendingFailure('x')), isFalse);
      expect(canForceDelete(const TopUpsPendingFailure('x')), isFalse);
      expect(canForceDelete(const NetworkFailure('x')), isFalse);
      expect(canForceDelete(null), isFalse);
    });
  });

  group('DeleteAccountController', () {
    late _FakeApi api;
    late ProviderContainer container;

    setUp(() {
      api = _FakeApi();
      container = ProviderContainer(
        overrides: [
          profileRemoteDataSourceProvider
              .overrideWithValue(ProfileRemoteDataSource(api.dio)),
        ],
      );
    });
    tearDown(() => container.dispose());

    DeleteAccountController ctrl() =>
        container.read(deleteAccountControllerProvider.notifier);
    DeleteAccountState state() => container.read(deleteAccountControllerProvider);

    test('ORDERS_IN_FLIGHT refusal is forceable; forced retry succeeds',
        () async {
      api.errorCode = 'ORDERS_IN_FLIGHT';
      expect(await ctrl().submit(), isFalse);
      expect(state().failure, isA<OrdersInFlightFailure>());
      expect(canForceDelete(state().failure), isTrue);
      expect(api.requests.last.data, isNull);

      api.errorCode = null;
      expect(await ctrl().submit(force: true), isTrue);
      expect(api.requests.last.data, {'force': true});
      expect(state().failure, isNull);
      expect(state().submitting, isFalse);
    });

    test('PAYMENTS_PENDING / TOPUPS_PENDING are not forceable, even forced',
        () async {
      for (final code in ['PAYMENTS_PENDING', 'TOPUPS_PENDING']) {
        api.errorCode = code;
        expect(await ctrl().submit(force: true), isFalse);
        expect(canForceDelete(state().failure), isFalse);
      }
      expect(state().failure, isA<TopUpsPendingFailure>());
    });
  });
}
