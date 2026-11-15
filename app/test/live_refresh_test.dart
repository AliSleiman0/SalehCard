import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/widgets/live_refresh.dart';

void main() {
  testWidgets('LiveRefresh calls onRefresh on every tick and stops when removed', (tester) async {
    var calls = 0;
    await tester.pumpWidget(LiveRefresh(
      interval: const Duration(seconds: 1),
      onRefresh: () => calls++,
      child: const SizedBox(),
    ));
    await tester.pump(const Duration(seconds: 3));
    expect(calls, 3);

    await tester.pumpWidget(const SizedBox());
    await tester.pump(const Duration(seconds: 3));
    expect(calls, 3);
  });

  testWidgets('LiveRefresh pauses in the background and refreshes on resume', (tester) async {
    var calls = 0;
    await tester.pumpWidget(LiveRefresh(
      interval: const Duration(seconds: 1),
      onRefresh: () => calls++,
      child: const SizedBox(),
    ));
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.hidden);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
    await tester.pump(const Duration(seconds: 3));
    expect(calls, 0);

    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.hidden);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    expect(calls, 1);
    await tester.pump(const Duration(seconds: 1));
    expect(calls, 2);
  });
}
