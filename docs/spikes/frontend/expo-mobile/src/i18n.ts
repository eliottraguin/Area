export const messages = {
  en: { title: 'My workflows', on: 'Enabled', off: 'Disabled', success: 'Last run: success', failed: 'Last run: failed', never: 'Never run', error: 'Cannot load workflows' },
  fr: { title: 'Mes workflows', on: 'Activé', off: 'Désactivé', success: 'Dernière exécution : succès', failed: 'Dernière exécution : échec', never: 'Jamais exécuté', error: 'Chargement impossible' },
} as const;
export type Lang = keyof typeof messages;
