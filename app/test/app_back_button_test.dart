import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:salehcard_app/core/widgets/app_back_button.dart';

/// Minimal screen: an AppBar with [AppBackButton] and the route name as body.
Widget _screen(String name, {String fallback = '/parent'}) => Scaffold(
      appBar: AppBar(leading: AppBackButton(fallback: fallback)),
      body: Text(name),
    );

GoRouter _router(String initial) => GoRouter(
      initialLocation: initial,
      routes: [
        GoRoute(path: '/parent', builder: (c, s) => const Text('parent')),
        GoRoute(path: '/list', builder: (c, s) => _screen('list')),
        GoRoute(path: '/detail', builder: (c, s) => _screen('detail')),
      ],
    );

void main() {
  testWidgets('pops to the previous screen when there is one', (tester) async {
    final router = _router('/list');
    await tester.pumpWidget(MaterialApp.router(routerConfig: router));
    router.push('/detail');
    await tester.pumpAndSettle();
    expect(find.text('detail'), findsOneWidget);

    await tester.tap(find.byType(BackButton));
    await tester.pumpAndSettle();
    expect(find.text('list'), findsOneWidget);
  });

  testWidgets('falls back to the parent when it is the stack root (reached via go)',
      (tester) async {
    final router = _router('/detail');
    await tester.pumpWidget(MaterialApp.router(routerConfig: router));
    await tester.pumpAndSettle();
    // Root of the stack: the implicit AppBar arrow would be missing here.
    expect(find.byType(BackButton), findsOneWidget);

    await tester.tap(find.byType(BackButton));
    await tester.pumpAndSettle();
    expect(find.text('parent'), findsOneWidget);
  });
}
