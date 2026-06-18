import { useCallback, useEffect, useState } from 'react';
import type { Device } from '../types/device';

const ORDER_KEY = 'emb:device-order';

function loadOrder(): string[] {
  try {
    const raw = localStorage.getItem(ORDER_KEY);
    return raw ? (JSON.parse(raw) as string[]) : [];
  } catch {
    return [];
  }
}

// useCardOrder keeps a user-defined device-card order in localStorage (per
// browser). Devices not yet in the saved order keep their server order and sort
// after the ordered ones, so newly added devices simply appear at the end.
export function useCardOrder(devices: Device[]): {
  ordered: Device[];
  move: (draggedId: string, targetId: string) => void;
} {
  const [order, setOrder] = useState<string[]>(loadOrder);

  useEffect(() => {
    localStorage.setItem(ORDER_KEY, JSON.stringify(order));
  }, [order]);

  const index = new Map(order.map((id, i) => [id, i]));
  const ordered = [...devices].sort((a, b) => {
    const ai = index.has(a.id) ? (index.get(a.id) as number) : Infinity;
    const bi = index.has(b.id) ? (index.get(b.id) as number) : Infinity;
    return ai - bi; // stable for equal (unsaved) entries
  });

  const move = useCallback(
    (draggedId: string, targetId: string) => {
      if (draggedId === targetId) return;
      // Rebuild the order from the currently displayed sequence so the saved
      // order always reflects what the user sees.
      const ids = ordered.map(d => d.id);
      const from = ids.indexOf(draggedId);
      const to = ids.indexOf(targetId);
      if (from === -1 || to === -1) return;
      ids.splice(to, 0, ids.splice(from, 1)[0]);
      setOrder(ids);
    },
    [ordered],
  );

  return { ordered, move };
}
