import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/money.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../domain/entities/wallet.dart';
import '../providers.dart';

/// Top-up screen: preset amount chips + a custom amount field, a card/USDT
/// method selector, and a "Top up $X" CTA. Submits via `POST /wallet/topups`.
///
/// NOTE: both `card` and `usdt` top-ups are mock-approved and credited
/// immediately in dev — there is no pending/poll state. On success we just pop
/// back to the wallet, which refreshes (TopUpController invalidates the wallet).
class TopUpScreen extends ConsumerStatefulWidget {
  const TopUpScreen({super.key});

  @override
  ConsumerState<TopUpScreen> createState() => _TopUpScreenState();
}

class _TopUpScreenState extends ConsumerState<TopUpScreen> {
  static const List<double> _presets = [25, 50, 100, 250];

  final _customController = TextEditingController();
  double? _selectedPreset = 50;
  String _method = 'card'; // card | usdt

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) ref.read(topUpControllerProvider.notifier).reset();
    });
  }

  @override
  void dispose() {
    _customController.dispose();
    super.dispose();
  }

  /// The effective amount: a typed custom value takes precedence, else the chip.
  double get _amount {
    final custom = double.tryParse(_customController.text.trim());
    if (custom != null && custom > 0) return custom;
    return _selectedPreset ?? 0;
  }

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context);
    final amount = _amount;
    if (amount <= 0) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.topUpAmount)),
      );
      return;
    }
    final tx = await ref.read(topUpControllerProvider.notifier).submit(
          TopUpInput(amount: amount, method: _method),
        );
    if (!mounted) return;
    if (tx != null) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.topUpSuccess(formatUsd(tx.amount.abs())))),
      );
      context.pop();
    } else {
      final failure = ref.read(topUpControllerProvider).failure;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(failure?.message.isNotEmpty == true
              ? failure!.message
              : l10n.paymentFailed),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final submitting = ref.watch(topUpControllerProvider).submitting;
    final amount = _amount;

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        title: Text(l10n.topUpCta),
      ),
      body: Column(
        children: [
          Expanded(
            child: ListView(
              padding: const EdgeInsets.fromLTRB(20, 18, 20, 24),
              children: [
                Text(
                  l10n.topUpAmount,
                  style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w800,
                      color: colors.text),
                ),
                const SizedBox(height: 12),
                Wrap(
                  spacing: 12,
                  runSpacing: 12,
                  children: [
                    for (final preset in _presets)
                      _AmountChip(
                        label: formatUsd(preset),
                        selected: _selectedPreset == preset &&
                            _customController.text.trim().isEmpty,
                        onTap: () {
                          FocusScope.of(context).unfocus();
                          setState(() {
                            _selectedPreset = preset;
                            _customController.clear();
                          });
                        },
                      ),
                  ],
                ),
                const SizedBox(height: 16),
                _CustomAmountField(
                  controller: _customController,
                  onChanged: (_) => setState(() {}),
                ),
                const SizedBox(height: 24),
                Text(
                  l10n.topUpVia,
                  style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w800,
                      color: colors.text),
                ),
                const SizedBox(height: 12),
                _MethodTile(
                  selected: _method == 'card',
                  icon: Icons.credit_card_rounded,
                  iconGradient: true,
                  title: l10n.payCardTitle,
                  subtitle: l10n.payCardSub,
                  onTap: () => setState(() => _method = 'card'),
                ),
                const SizedBox(height: 10),
                _MethodTile(
                  selected: _method == 'usdt',
                  icon: Icons.currency_exchange_rounded,
                  iconColor: const Color(0xFF26A17B),
                  title: l10n.payUsdtTitle,
                  subtitle: l10n.payUsdtSub,
                  onTap: () => setState(() => _method = 'usdt'),
                ),
              ],
            ),
          ),
          _TopUpBar(
            label: '${l10n.topUpCta}  ·  ${formatUsd(amount)}',
            submitting: submitting,
            enabled: amount > 0,
            colors: colors,
            onTap: _submit,
          ),
        ],
      ),
    );
  }
}

class _AmountChip extends StatelessWidget {
  const _AmountChip({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        width: 96,
        padding: const EdgeInsets.symmetric(vertical: 16),
        alignment: Alignment.center,
        decoration: BoxDecoration(
          gradient: selected ? AppTokens.brandGradient : null,
          color: selected ? null : colors.surface,
          border: selected ? null : Border.all(color: colors.border),
          borderRadius: BorderRadius.circular(AppTokens.rMd),
        ),
        child: Text(
          label,
          style: TextStyle(
            fontSize: 15,
            fontWeight: FontWeight.w800,
            color: selected ? Colors.white : colors.text,
          ),
        ),
      ),
    );
  }
}

class _CustomAmountField extends StatelessWidget {
  const _CustomAmountField({required this.controller, required this.onChanged});

  final TextEditingController controller;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppTokens.rMd),
      borderSide: BorderSide(color: colors.border),
    );
    return TextField(
      controller: controller,
      onChanged: onChanged,
      keyboardType: const TextInputType.numberWithOptions(decimal: true),
      inputFormatters: [
        FilteringTextInputFormatter.allow(RegExp(r'[0-9.]')),
      ],
      style: TextStyle(fontSize: 15, color: colors.text),
      cursorColor: AppTokens.brand1,
      decoration: InputDecoration(
        filled: true,
        fillColor: colors.surface,
        prefixIcon: Icon(Icons.attach_money_rounded, color: colors.textFaint),
        hintText: AppLocalizations.of(context).amountLabel,
        hintStyle: TextStyle(color: colors.textFaint),
        contentPadding:
            const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
        border: border,
        enabledBorder: border,
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppTokens.rMd),
          borderSide: const BorderSide(color: AppTokens.brand1, width: 1.6),
        ),
      ),
    );
  }
}

class _MethodTile extends StatelessWidget {
  const _MethodTile({
    required this.selected,
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
    this.iconGradient = false,
    this.iconColor,
  });

  final bool selected;
  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;
  final bool iconGradient;
  final Color? iconColor;

  @override
  Widget build(BuildContext context) {
    final colors = context.colors;
    return GestureDetector(
      onTap: onTap,
      behavior: HitTestBehavior.opaque,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 13),
        decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: BorderRadius.circular(AppTokens.rMd),
          border: Border.all(
            color: selected ? AppTokens.cta : colors.border,
            width: selected ? 2 : 1,
          ),
        ),
        child: Row(
          children: [
            Container(
              width: 38,
              height: 38,
              decoration: BoxDecoration(
                gradient: iconGradient ? AppTokens.brandGradient : null,
                color: iconGradient
                    ? null
                    : (iconColor ?? AppTokens.accent).withValues(alpha: 0.16),
                borderRadius: BorderRadius.circular(11),
              ),
              child: Icon(icon,
                  size: 19, color: iconGradient ? Colors.white : iconColor),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(title,
                      style: TextStyle(
                          fontSize: 14.5,
                          fontWeight: FontWeight.w800,
                          color: colors.text)),
                  const SizedBox(height: 1),
                  Text(subtitle,
                      style: TextStyle(
                          fontSize: 12.5,
                          fontWeight: FontWeight.w600,
                          color: colors.textDim)),
                ],
              ),
            ),
            _Radio(selected: selected),
          ],
        ),
      ),
    );
  }
}

class _Radio extends StatelessWidget {
  const _Radio({required this.selected});

  final bool selected;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 22,
      height: 22,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        border: Border.all(
          color: selected ? AppTokens.cta : context.colors.borderStrong,
          width: 2,
        ),
      ),
      child: selected
          ? Center(
              child: Container(
                width: 10,
                height: 10,
                decoration: const BoxDecoration(
                  shape: BoxShape.circle,
                  color: AppTokens.cta,
                ),
              ),
            )
          : null,
    );
  }
}

class _TopUpBar extends StatelessWidget {
  const _TopUpBar({
    required this.label,
    required this.submitting,
    required this.enabled,
    required this.colors,
    required this.onTap,
  });

  final String label;
  final bool submitting;
  final bool enabled;
  final AppColors colors;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.fromLTRB(
          18, 13, 18, 13 + MediaQuery.of(context).padding.bottom),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border(top: BorderSide(color: colors.border)),
      ),
      child: FilledButton(
        onPressed: (submitting || !enabled) ? null : onTap,
        style: FilledButton.styleFrom(
          minimumSize: const Size.fromHeight(54),
          backgroundColor: AppTokens.cta,
          disabledBackgroundColor: AppTokens.cta.withValues(alpha: 0.5),
          foregroundColor: Colors.white,
          disabledForegroundColor: Colors.white,
          shape: const StadiumBorder(),
          elevation: 8,
          shadowColor: AppTokens.cta.withValues(alpha: 0.3),
          textStyle:
              const TextStyle(fontSize: 16.5, fontWeight: FontWeight.w800),
        ),
        child: submitting
            ? const AppSpinner(color: Colors.white, size: 22, stroke: 2.5)
            : Text(label),
      ),
    );
  }
}
