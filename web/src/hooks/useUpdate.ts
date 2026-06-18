import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';

const VERSION_KEY = ['version'];

// useVersion polls the running version + update-availability status. The check
// against GitHub is cached server-side, so a frequent client poll is cheap.
export function useVersion() {
  return useQuery({
    queryKey: VERSION_KEY,
    queryFn: api.getVersion,
    refetchInterval: 30 * 60 * 1000, // re-read every 30 min
    staleTime: 5 * 60 * 1000,
  });
}

export function useApplyUpdate() {
  return useMutation({
    mutationFn: api.applyUpdate,
  });
}

// useCheckUpdate forces an immediate re-check against GitHub Releases and writes
// the fresh status into the version cache, so the About page, the version badge,
// and the update pop-up all react without an app restart.
export function useCheckUpdate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: api.checkUpdate,
    onSuccess: (status) => {
      qc.setQueryData(VERSION_KEY, status);
    },
  });
}
