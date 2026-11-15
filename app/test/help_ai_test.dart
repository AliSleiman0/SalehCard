import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/i18n/arb/app_localizations.dart';
import 'package:salehcard_app/core/theme/app_theme.dart';
import 'package:salehcard_app/features/help/data/chatbot_repository.dart';
import 'package:salehcard_app/features/help/domain/chat_logic.dart';
import 'package:salehcard_app/features/help/presentation/help_sheet.dart';

/// A Dio that never leaves the process: /chatbot/ask answers with [askStatus]
/// (200 → [askBody]); every request is recorded.
class _FakeApi {
  int askStatus = 200;
  Map<String, dynamic> askBody = {
    'logId': 'log1',
    'answer': 'Open Wallet and tap Top up.',
    'answered': true,
    'showSupport': false,
  };
  final requests = <RequestOptions>[];

  late final Dio dio = Dio(BaseOptions(baseUrl: 'http://test'))
    ..interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requests.add(options);
          final isAsk = options.path == '/chatbot/ask';
          final status = isAsk ? askStatus : 200;
          if (status != 200) {
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.badResponse,
                response: Response(
                  requestOptions: options,
                  statusCode: status,
                  data: {
                    'success': false,
                    'error': {'code': 'X', 'message': 'x'},
                  },
                ),
              ),
            );
            return;
          }
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: {
                'success': true,
                'data': isAsk ? askBody : {'reported': true},
              },
            ),
          );
        },
      ),
    );
}

Widget _app(_FakeApi api, {required bool ai}) => ProviderScope(
  overrides: [
    chatbotProvider.overrideWith(
      (ref) async => ChatbotData(
        support: const ChatSupport(whatsapp: '+961 70 123 456'),
        ai: ai,
      ),
    ),
    chatbotRepositoryProvider.overrideWithValue(ChatbotRepository(api.dio)),
  ],
  child: MaterialApp(
    theme: AppTheme.light(),
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: AppLocalizations.supportedLocales,
    home: const Scaffold(body: HelpSheet()),
  ),
);

Future<void> _send(WidgetTester tester, String text) async {
  await tester.enterText(find.byType(TextField), text);
  await tester.testTextInput.receiveAction(TextInputAction.send);
  await tester.pumpAndSettle();
}

void main() {
  test('parses the ai flag and an ask result', () {
    expect(ChatbotData.fromJson({'ai': true}).ai, isTrue);
    expect(ChatbotData.fromJson({}).ai, isFalse);
    final r = AskResult.fromJson({
      'logId': 'l',
      'answer': 'a',
      'answered': true,
      'showSupport': false,
    });
    expect([r.logId, r.answer, r.answered], ['l', 'a', true]);
  });

  test('askFailure maps 429 to limited, everything else to unavailable', () {
    DioException err(int? status) => DioException(
      requestOptions: RequestOptions(),
      response: status == null
          ? null
          : Response(requestOptions: RequestOptions(), statusCode: status),
    );
    expect(askFailure(err(429)), AskFailure.limited);
    expect(askFailure(err(503)), AskFailure.unavailable);
    expect(askFailure(err(null)), AskFailure.unavailable);
    expect(askFailure(StateError('x')), AskFailure.unavailable);
  });

  test('askHistory keeps the last turns, oldest first', () {
    final turns = [
      for (var i = 0; i < 9; i++) (fromUser: i.isEven, text: 't$i'),
    ];
    final h = askHistory(turns);
    expect(h.length, 6);
    expect(h.first.content, 't3');
    expect(h.first.role, 'assistant');
    expect(h.last.content, 't8');
    expect(h.last.role, 'user');
  });

  testWidgets('AI on: typed question gets the AI answer and can be reported', (
    tester,
  ) async {
    final api = _FakeApi();
    await tester.pumpWidget(_app(api, ai: true));
    await tester.pumpAndSettle();

    await _send(tester, 'how do I add balance?');
    expect(find.text('Open Wallet and tap Top up.'), findsOneWidget);
    expect(find.textContaining('AI answer'), findsOneWidget);
    final ask = api.requests.singleWhere((r) => r.path == '/chatbot/ask');
    final body = ask.data as Map<String, dynamic>;
    expect(body['question'], 'how do I add balance?');
    expect(body['lang'], 'en');
    expect(body['platform'], 'android');

    await tester.tap(find.byKey(const ValueKey('help-ai-report')));
    await tester.pumpAndSettle();
    expect(
      api.requests.where((r) => r.path == '/chatbot/ask/log1/report'),
      hasLength(1),
    );
    expect(find.textContaining("we'll review"), findsOneWidget);
  });

  testWidgets('AI on, limit reached: says so', (tester) async {
    final api = _FakeApi()..askStatus = 429;
    await tester.pumpWidget(_app(api, ai: true));
    await tester.pumpAndSettle();
    await _send(tester, 'wallet');
    expect(find.textContaining('reached the limit'), findsOneWidget);
  });

  testWidgets('AI on, not answerable: shows the fallback and the contacts', (
    tester,
  ) async {
    final api = _FakeApi()
      ..askBody = {
        'logId': 'log2',
        'answer': 'Sorry, I can only answer questions about using the app.',
        'answered': false,
        'showSupport': true,
      };
    await tester.pumpWidget(_app(api, ai: true));
    await tester.pumpAndSettle();
    await _send(tester, 'write me a poem');
    expect(find.textContaining('I can only answer'), findsOneWidget);
    expect(find.textContaining('AI answer'), findsNothing);
    expect(find.text('WhatsApp'), findsOneWidget);
  });

  testWidgets('AI unavailable: says so and points at support', (tester) async {
    final api = _FakeApi()..askStatus = 503;
    await tester.pumpWidget(_app(api, ai: true));
    await tester.pumpAndSettle();
    await _send(tester, 'wallet');
    expect(find.textContaining("isn't available right now"), findsOneWidget);
    expect(find.textContaining('AI answer'), findsNothing);
    expect(find.text('WhatsApp'), findsOneWidget);
  });

  testWidgets('AI off: no input, only the support contacts', (tester) async {
    final api = _FakeApi();
    await tester.pumpWidget(_app(api, ai: false));
    await tester.pumpAndSettle();
    expect(find.byType(TextField), findsNothing);
    expect(find.textContaining('support team is here'), findsOneWidget);
    expect(find.text("We're here to help"), findsOneWidget);
    expect(find.text('WhatsApp'), findsOneWidget);
    expect(api.requests.where((r) => r.path == '/chatbot/ask'), isEmpty);
  });
}
