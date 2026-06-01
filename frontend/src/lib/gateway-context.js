import { useOutletContext } from "react-router-dom";

export function useGateway() {
  return useOutletContext();
}
