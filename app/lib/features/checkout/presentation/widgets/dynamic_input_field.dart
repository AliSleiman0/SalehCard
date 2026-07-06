import 'package:flutter/material.dart';

import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/theme/app_tokens.dart';
import '../../../catalog/domain/entities/product.dart';

/// Renders one dynamic product [InputField] (`text | amount | quantity |
/// select`) at checkout. Sensitive fields are masked. The parent owns the value:
/// text-like types via [controller], `select` via [value] + [onChanged].
class DynamicInputField extends StatelessWidget {
  const DynamicInputField({
    super.key,
    required this.field,
    required this.localeCode,
    this.controller,
    this.value,
    this.onChanged,
    this.errorText,
    this.showRequiredBadge = true,
  });

  final InputField field;
  final String localeCode;
  final TextEditingController? controller;
  final String? value;
  final ValueChanged<String>? onChanged;
  final String? errorText;
  final bool showRequiredBadge;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final label = field.label.resolve(localeCode);
    final hasError = errorText != null;

    final border = OutlineInputBorder(
      borderRadius: BorderRadius.circular(AppTokens.rMd),
      borderSide: BorderSide(
        color: hasError ? AppTokens.danger : colors.border,
      ),
    );

    Widget input;
    if (field.type == 'select') {
      final options = field.constraints?.options ?? const <String>[];
      input = DropdownButtonFormField<String>(
        initialValue:
            (value != null && options.contains(value)) ? value : null,
        isExpanded: true,
        items: [
          for (final opt in options)
            DropdownMenuItem(value: opt, child: Text(opt)),
        ],
        onChanged: (v) => onChanged?.call(v ?? ''),
        dropdownColor: colors.surface,
        style: TextStyle(fontSize: 15, color: colors.text),
        decoration: InputDecoration(
          filled: true,
          fillColor: colors.surface,
          contentPadding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          border: border,
          enabledBorder: border,
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(AppTokens.rMd),
            borderSide: BorderSide(
              color: hasError ? AppTokens.danger : AppTokens.brand1,
              width: 1.6,
            ),
          ),
        ),
      );
    } else {
      input = TextField(
        controller: controller,
        obscureText: field.sensitive,
        keyboardType: _keyboardType(field.type, field.key),
        onChanged: onChanged,
        style: TextStyle(fontSize: 15, color: colors.text),
        cursorColor: AppTokens.brand1,
        decoration: InputDecoration(
          filled: true,
          fillColor: colors.surface,
          hintText: l10n.enterValue(label),
          hintStyle: TextStyle(color: colors.textFaint),
          contentPadding:
              const EdgeInsets.symmetric(horizontal: 16, vertical: 15),
          border: border,
          enabledBorder: border,
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(AppTokens.rMd),
            borderSide: BorderSide(
              color: hasError ? AppTokens.danger : AppTokens.brand1,
              width: 1.6,
            ),
          ),
        ),
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Flexible(
              child: Text(
                label.isEmpty ? field.key : label,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  color: colors.textDim,
                ),
              ),
            ),
            if (showRequiredBadge) ...[
              const SizedBox(width: 7),
              _RequiredBadge(label: l10n.requiredBadge),
            ],
          ],
        ),
        const SizedBox(height: 8),
        input,
        if (hasError) ...[
          const SizedBox(height: 6),
          Text(
            errorText!,
            style: const TextStyle(
              fontSize: 12.5,
              fontWeight: FontWeight.w500,
              color: AppTokens.danger,
            ),
          ),
        ],
      ],
    );
  }

  TextInputType _keyboardType(String type, String key) {
    if (key == 'phone') return TextInputType.phone;
    switch (type) {
      case 'amount':
        return const TextInputType.numberWithOptions(decimal: true);
      case 'quantity':
        return TextInputType.number;
      default:
        return TextInputType.text;
    }
  }
}

class _RequiredBadge extends StatelessWidget {
  const _RequiredBadge({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
      decoration: BoxDecoration(
        color: AppTokens.brand2.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(AppTokens.rPill),
      ),
      child: Text(
        label,
        style: const TextStyle(
          fontSize: 10.5,
          fontWeight: FontWeight.w700,
          color: AppTokens.brand2,
        ),
      ),
    );
  }
}
