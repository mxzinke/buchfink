/**
 * German locale formatters for number and date formatting
 */

export function formatCurrency(amount: number, currency: string = 'EUR'): string {
  if (isNaN(amount)) return '0,00 €';
  
  const formatted = new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount);

  return formatted;
}

export function formatDate(dateStr: string): string {
  if (!dateStr) return '—';
  try {
    const parts = dateStr.split('T')[0].split('-');
    if (parts.length === 3) {
      return `${parts[2]}.${parts[1]}.${parts[0]}`;
    }
    const d = new Date(dateStr);
    return new Intl.DateTimeFormat('de-DE').format(d);
  } catch {
    return dateStr;
  }
}

export function formatShortHash(hash: string): string {
  if (!hash) return '—';
  if (hash.length <= 12) return hash;
  return `${hash.substring(0, 6)}...${hash.substring(hash.length - 6)}`;
}

export function formatPercent(rate: number): string {
  return `${Math.round(rate * 100)} %`;
}
