import { apiRequest } from './api';

export type ServicesResponse = {
  services: string[];
};

export function fetchServices(): Promise<ServicesResponse> {
  return apiRequest<ServicesResponse>('/api/services');
}
