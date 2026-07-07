import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../checkout/presentation/providers.dart' show ordersProvider;
import '../../../wallet/presentation/providers.dart'
    show walletProvider, topUpRequestsProvider;
import '../../domain/entities/payment_intent.dart';
import '../providers.dart';

/// Redirect-and-poll screen for a Whish payment. It opens the hosted Whish
/// "collect" page in the external browser (Whish redirects can break inside a
/// webview) and — mirroring the USDT deposit screen — polls
/// `GET /payments/intents/{id}` every few seconds (lifecycle-aware) until the
/// intent is confirmed, expired, or failed. Ground truth is the backend re-poll
/// in the Whish callback, so the browser landing page is irrelevant here.
///
/// Reached with the freshly-created [PaymentIntent] as the router `extra`.
class WhishRedirectScreen extends ConsumerStatefulWidget {
  const WhishRedirectScreen({super.key, required this.intent});

  final PaymentIntent intent;

  @override
  ConsumerState<WhishRedirectScreen> createState() =>
      _WhishRedirectScreenState();
}

class _WhishRedirectScreenState extends ConsumerState<WhishRedirectScreen>
    with WidgetsBindingObserver {
  static const _pollInterval = Duration(seconds: 7);
  Timer? _pollTimer;
  bool _navigated = false;
  bool _opened = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _startPolling();
    // Auto-open the hosted page on first frame.
    WidgetsBinding.instance.addPostFrameCallback((_) => _openWhish());
  }

  @override
  void dispose() {
    _stopPolling();
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _refresh();
      _startPolling();
    } else {
      _stopPolling();
    }
  }

  void _startPolling() =>
      _pollTimer ??= Timer.periodic(_pollInterval, (_) => _refresh());

  void _stopPolling() {
    _pollTimer?.cancel();
    _pollTimer = null;
  }

  void _refresh() => ref.invalidate(paymentIntentProvider(widget.intent.id));

  Future<void> _openWhish() async {
    _opened = true;
    final url = Uri.tryParse(widget.intent.redirectUrl);
    if (url == null) return;
    final ok = await launchUrl(url, mode: LaunchMode.externalApplication);
    if (!ok && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Could not open Whish. Please try again.')),
      );
    }
  }

  void _onConfirmed(PaymentIntent intent) {
    if (_navigated) return;
    _navigated = true;
    _stopPolling();
    ref.invalidate(walletProvider);
    ref.invalidate(topUpRequestsProvider);
    ref.invalidate(ordersProvider);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (intent.purpose == PaymentPurpose.order && intent.orderId != null) {
        context.pushReplacement('/orders/${intent.orderId}');
      } else {
        context.go('/wallet');
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final async = ref.watch(paymentIntentProvider(widget.intent.id));
    final intent = async.asData?.value ?? widget.intent;

    if (intent.isConfirmed) {
      _onConfirmed(intent);
      return const _TerminalPanel(
        icon: Icons.check_circle_rounded,
        color: AppTokens.accent,
        title: 'Payment confirmed',
        body: 'Your Whish payment was confirmed.',
      );
    }
    if (intent.isExpired || intent.isFailed) {
      return _TerminalPanel(
        icon: Icons.cancel_rounded,
        color: AppTokens.danger,
        title: intent.isExpired ? 'Payment expired' : 'Payment not completed',
        body: intent.isExpired
            ? 'The payment window expired before Whish completed the payment. No money was taken.'
            : 'Your Whish payment did not go through. No money was taken; you can try again.',
        actionLabel: 'Done',
        onAction: () => context.pop(),
      );
    }

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: const Text('Pay with Whish'),
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(24, 28, 24, 28),
        children: [
          Icon(Icons.phone_iphone_rounded, size: 56, color: colors.textDim),
          const SizedBox(height: 18),
          Text(
            'Complete your \$${intent.amountUsd.toStringAsFixed(2)} payment in Whish',
            textAlign: TextAlign.center,
            style: TextStyle(
              fontSize: 17,
              fontWeight: FontWeight.w800,
              color: colors.text,
            ),
          ),
          const SizedBox(height: 10),
          Text(
            _opened
                ? 'Finish the payment in Whish, then return here — this screen updates automatically.'
                : 'Opening Whish…',
            textAlign: TextAlign.center,
            style: TextStyle(
              fontSize: 13.5,
              fontWeight: FontWeight.w600,
              color: colors.textDim,
            ),
          ),
          const SizedBox(height: 26),
          FilledButton.icon(
            onPressed: _openWhish,
            icon: const Icon(Icons.open_in_new_rounded, size: 18),
            style: FilledButton.styleFrom(
              minimumSize: const Size.fromHeight(52),
              backgroundColor: AppTokens.cta,
              foregroundColor: Colors.white,
              shape: const StadiumBorder(),
              textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w800),
            ),
            label: const Text('Continue to Whish'),
          ),
          const SizedBox(height: 24),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const AppSpinner(size: 16),
              const SizedBox(width: 10),
              Text(
                'Waiting for payment…',
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  color: colors.textFaint,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _TerminalPanel extends StatelessWidget {
  const _TerminalPanel({
    required this.icon,
    required this.color,
    required this.title,
    required this.body,
    this.actionLabel,
    this.onAction,
  });

  final IconData icon;
  final Color color;
  final String title;
  final String body;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return Scaffold(
      backgroundColor: colors.bg,
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(28),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 64, color: color),
              const SizedBox(height: 18),
              Text(
                title,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w800,
                  color: colors.text,
                ),
              ),
              const SizedBox(height: 10),
              Text(
                body,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                  color: colors.textDim,
                ),
              ),
              if (actionLabel != null) ...[
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: onAction,
                  style: FilledButton.styleFrom(
                    minimumSize: const Size(180, 50),
                    backgroundColor: AppTokens.cta,
                    foregroundColor: Colors.white,
                    shape: const StadiumBorder(),
                  ),
                  child: Text(actionLabel!),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
