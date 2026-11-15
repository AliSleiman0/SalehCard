import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// AppBar back control that never strands the user: pops when there is a
/// screen beneath, otherwise goes to [fallback] (the screen's logical parent).
///
/// A full-screen route reached with `context.go` (order-success → order,
/// order → orders list, USDT deposit → wallet, …) is the root of the stack, so
/// the implicit AppBar arrow is missing and Android's system back would close
/// the app. Put this in `AppBar.leading` on any screen that can be reached that
/// way. Uses Material's [BackButton], so the arrow mirrors in RTL and the
/// tooltip is localized.
class AppBackButton extends StatelessWidget {
  const AppBackButton({super.key, required this.fallback});

  /// Route to go to when there is nothing to pop (e.g. '/account', '/home').
  final String fallback;

  @override
  Widget build(BuildContext context) {
    return BackButton(
      onPressed: () =>
          context.canPop() ? context.pop() : context.go(fallback),
    );
  }
}
