import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/format/date_time.dart';
import '../../../../core/i18n/arb/app_localizations.dart';
import '../../../../core/theme/app_colors.dart';
import '../../../../core/widgets/app_back_button.dart';
import '../../../../core/widgets/app_spinner.dart';
import '../../../../core/widgets/empty_state.dart';
import '../../../../core/widgets/status_badge.dart';
import '../../data/complaint_repository.dart';
import '../complaint_labels.dart';

/// "My complaints": number, subject, status and last activity, newest first.
class ComplaintsListScreen extends ConsumerWidget {
  const ComplaintsListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final colors = context.colors;
    final async = ref.watch(complaintsProvider);

    return Scaffold(
      backgroundColor: colors.bg,
      appBar: AppBar(
        backgroundColor: colors.topbar,
        // Normally pushed from the Menu tab; complaint detail's fallback reaches
        // it via go as the stack root.
        leading: const AppBackButton(fallback: '/account'),
        title: Text(l10n.complaintsTitle),
      ),
      floatingActionButton: (async.value?.isNotEmpty ?? false)
          ? FloatingActionButton.extended(
              onPressed: () => context.push('/complaints/new'),
              icon: const Icon(Icons.add_rounded),
              label: Text(l10n.complaintSubmit),
            )
          : null,
      body: async.when(
        loading: () => const LoadingView(),
        error: (_, _) => Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(l10n.complaintsLoadFailed, textAlign: TextAlign.center),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () => ref.invalidate(complaintsProvider),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
        ),
        data: (items) {
          if (items.isEmpty) {
            return EmptyState(
              icon: Icons.support_agent_rounded,
              title: l10n.complaintsEmptyTitle,
              message: l10n.complaintsEmptySub,
              actionLabel: l10n.complaintSubmit,
              onAction: () => context.push('/complaints/new'),
            );
          }
          return RefreshIndicator(
            onRefresh: () async {
              ref.invalidate(complaintsProvider);
              await ref.read(complaintsProvider.future);
            },
            child: ListView.separated(
              padding: const EdgeInsets.only(top: 8, bottom: 96),
              itemCount: items.length,
              separatorBuilder: (_, _) =>
                  Divider(height: 1, color: colors.border),
              itemBuilder: (_, i) {
                final c = items[i];
                final (label, color) = complaintStatusLabel(c.status, l10n);
                return ListTile(
                  contentPadding: const EdgeInsets.symmetric(
                    horizontal: 18,
                    vertical: 6,
                  ),
                  title: Row(
                    children: [
                      Text(
                        c.ref,
                        textDirection: TextDirection.ltr,
                        style: TextStyle(
                          fontSize: 12.5,
                          fontWeight: FontWeight.w700,
                          color: colors.textFaint,
                        ),
                      ),
                      const SizedBox(width: 8),
                      StatusBadge(label: label, color: color),
                    ],
                  ),
                  subtitle: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const SizedBox(height: 4),
                      Text(
                        c.subject,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          fontSize: 14.5,
                          fontWeight: FontWeight.w800,
                          color: colors.text,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        '${l10n.complaintLastActivity}: ${formatDateTime(c.updatedAt)}',
                        style: TextStyle(fontSize: 12, color: colors.textFaint),
                      ),
                    ],
                  ),
                  trailing: const Icon(Icons.chevron_right_rounded),
                  onTap: () => context.push('/complaints/${c.id}'),
                );
              },
            ),
          );
        },
      ),
    );
  }
}
