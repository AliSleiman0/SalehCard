import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/i18n/arb/app_localizations.dart';
import 'package:salehcard_app/core/theme/app_theme.dart';
import 'package:salehcard_app/features/account/data/datasources/profile_remote_data_source.dart';
import 'package:salehcard_app/features/account/domain/password_rules.dart';
import 'package:salehcard_app/features/account/presentation/providers.dart';
import 'package:salehcard_app/features/account/presentation/screens/change_password_screen.dart';

/// A Dio that never leaves the process: records requests and answers with a
/// `{changed:true}` envelope, or a 400 carrying [errorCode].
class _FakeApi {
  String? errorCode;
  final requests = <RequestOptions>[];

  late final Dio dio = Dio(BaseOptions(baseUrl: 'http://test'))
    ..interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requests.add(options);
          final code = errorCode;
          if (code == null) {
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {
                  'success': true,
                  'data': {'changed': true},
                },
              ),
            );
            return;
          }
          handler.reject(
            DioException(
              requestOptions: options,
              type: DioExceptionType.badResponse,
              response: Response(
                requestOptions: options,
                statusCode: 400,
                data: {
                  'success': false,
                  'error': {'code': code, 'message': code},
                },
              ),
            ),
          );
        },
      ),
    );
}

Widget _app(_FakeApi api, {Locale locale = const Locale('en')}) =>
    ProviderScope(
      overrides: [
        profileRemoteDataSourceProvider.overrideWithValue(
          ProfileRemoteDataSource(api.dio),
        ),
      ],
      child: MaterialApp(
        locale: locale,
        theme: AppTheme.light(),
        localizationsDelegates: const [
          AppLocalizations.delegate,
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        supportedLocales: AppLocalizations.supportedLocales,
        home: const ChangePasswordScreen(),
      ),
    );

Future<void> _fill(
  WidgetTester tester,
  String current,
  String next,
  String confirm,
) async {
  await tester.enterText(
    find.descendant(
      of: find.byKey(const Key('pw-current')),
      matching: find.byType(TextField),
    ),
    current,
  );
  await tester.enterText(
    find.descendant(
      of: find.byKey(const Key('pw-new')),
      matching: find.byType(TextField),
    ),
    next,
  );
  await tester.enterText(
    find.descendant(
      of: find.byKey(const Key('pw-confirm')),
      matching: find.byType(TextField),
    ),
    confirm,
  );
  await tester.tap(find.byKey(const Key('pw-submit')));
  await tester.pumpAndSettle();
}

void main() {
  group('validateChangePassword', () {
    PasswordProblem? v(String c, String n, String k) =>
        validateChangePassword(current: c, next: n, confirm: k);

    test('accepts a valid change', () {
      expect(v('password123', 'newpass123', 'newpass123'), isNull);
    });

    test('requires the current password', () {
      expect(
        v('', 'newpass123', 'newpass123'),
        PasswordProblem.currentRequired,
      );
    });

    test('min 8 characters', () {
      expect(v('x', 'short1', 'short1'), PasswordProblem.tooShort);
      expect(v('x', '12345678', '12345678'), isNull);
    });

    test('max 72 counted in UTF-8 bytes, not characters', () {
      expect(v('x', 'a' * 72, 'a' * 72), isNull);
      expect(v('x', 'a' * 73, 'a' * 73), PasswordProblem.tooLong);
      // 37 Arabic letters = 74 bytes.
      expect(utf8ByteLength('ب' * 37), 74);
      expect(v('x', 'ب' * 37, 'ب' * 37), PasswordProblem.tooLong);
      expect(v('x', 'ب' * 36, 'ب' * 36), isNull);
    });

    test('confirmation must match', () {
      expect(v('x', 'newpass123', 'newpass124'), PasswordProblem.mismatch);
    });
  });

  test('passwordProblemForCode maps each server code', () {
    expect(
      passwordProblemForCode('WRONG_PASSWORD'),
      PasswordProblem.wrongCurrent,
    );
    expect(passwordProblemForCode('WEAK_PASSWORD'), PasswordProblem.tooShort);
    expect(
      passwordProblemForCode('PASSWORD_TOO_LONG'),
      PasswordProblem.tooLong,
    );
    expect(
      passwordProblemForCode('NO_PASSWORD_SET'),
      PasswordProblem.noPasswordSet,
    );
    expect(passwordProblemForCode('RATE_LIMITED'), PasswordProblem.rateLimited);
    expect(passwordProblemForCode('SOMETHING_ELSE'), isNull);
  });

  group('ChangePasswordScreen', () {
    testWidgets('a short password is blocked before any request', (
      tester,
    ) async {
      final api = _FakeApi();
      await tester.pumpWidget(_app(api));
      await _fill(tester, 'password123', 'short', 'short');
      expect(
        find.text('The new password must be at least 8 characters.'),
        findsOneWidget,
      );
      expect(api.requests, isEmpty);
    });

    testWidgets('a mismatched confirmation is blocked', (tester) async {
      final api = _FakeApi();
      await tester.pumpWidget(_app(api));
      await _fill(tester, 'password123', 'newpass123', 'newpass999');
      expect(find.text("The two new passwords don't match."), findsOneWidget);
      expect(api.requests, isEmpty);
    });

    testWidgets('WRONG_PASSWORD shows its own message', (tester) async {
      final api = _FakeApi()..errorCode = 'WRONG_PASSWORD';
      await tester.pumpWidget(_app(api));
      await _fill(tester, 'nope12345', 'newpass123', 'newpass123');
      expect(find.text('Your current password is incorrect.'), findsOneWidget);
      expect(api.requests.single.path, '/users/me/password');
      expect(api.requests.single.data, {
        'currentPassword': 'nope12345',
        'newPassword': 'newpass123',
      });
    });

    testWidgets('NO_PASSWORD_SET shows the SMS sign-in hint', (tester) async {
      final api = _FakeApi()..errorCode = 'NO_PASSWORD_SET';
      await tester.pumpWidget(_app(api));
      await _fill(tester, 'anything1', 'newpass123', 'newpass123');
      expect(
        find.text("This account doesn't have a password yet."),
        findsOneWidget,
      );
      expect(
        find.textContaining('signing in with an SMS code'),
        findsOneWidget,
      );
    });

    testWidgets('success clears the form and confirms', (tester) async {
      final api = _FakeApi();
      await tester.pumpWidget(_app(api));
      await _fill(tester, 'password123', 'newpass123', 'newpass123');
      expect(find.text('Password changed'), findsOneWidget);
      expect(find.byKey(const Key('pw-error')), findsNothing);
      final fields = tester.widgetList<TextField>(find.byType(TextField));
      expect(fields.every((f) => f.controller!.text.isEmpty), isTrue);
    });

    testWidgets('Arabic renders right-to-left', (tester) async {
      final api = _FakeApi();
      await tester.pumpWidget(_app(api, locale: const Locale('ar')));
      await tester.pumpAndSettle();
      expect(find.text('كلمة المرور الحالية'), findsOneWidget);
      final dir = Directionality.of(
        tester.element(find.byType(ChangePasswordScreen)),
      );
      expect(dir, TextDirection.rtl);
    });
  });

  test('controller provider is wired to the use case', () {
    final api = _FakeApi();
    final container = ProviderContainer(
      overrides: [
        profileRemoteDataSourceProvider.overrideWithValue(
          ProfileRemoteDataSource(api.dio),
        ),
      ],
    );
    addTearDown(container.dispose);
    expect(
      container.read(changePasswordControllerProvider).submitting,
      isFalse,
    );
  });
}
