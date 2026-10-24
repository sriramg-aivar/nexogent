import React from 'react'

const S = (p) => ({ width: 18, height: 18, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2, strokeLinecap: 'round', strokeLinejoin: 'round', ...p })

export const IconOverview = (p) => <svg {...S(p)}><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
export const IconFindings = (p) => <svg {...S(p)}><path d="M9 11l3 3L22 4"/><path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11"/></svg>
export const IconPath = (p) => <svg {...S(p)}><circle cx="5" cy="6" r="2"/><circle cx="19" cy="18" r="2"/><path d="M7 6h7a3 3 0 013 3v6"/></svg>
export const IconInventory = (p) => <svg {...S(p)}><path d="M21 16V8a2 2 0 00-1-1.7l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.7l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/><path d="M3.3 7L12 12l8.7-5M12 22V12"/></svg>
export const IconShield = (p) => <svg {...S(p)}><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
export const IconFile = (p) => <svg {...S(p)}><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><path d="M14 2v6h6"/></svg>
export const IconClock = (p) => <svg {...S(p)}><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></svg>
export const IconSettings = (p) => <svg {...S(p)}><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.6 1.6 0 00.3 1.8l.1.1a2 2 0 11-2.8 2.8l-.1-.1a1.6 1.6 0 00-2.7 1.1V21a2 2 0 11-4 0v-.1A1.6 1.6 0 006 19.4l-.1.1a2 2 0 11-2.8-2.8l.1-.1a1.6 1.6 0 00-1.1-2.7H2a2 2 0 110-4h.1A1.6 1.6 0 003.6 6l-.1-.1a2 2 0 112.8-2.8l.1.1a1.6 1.6 0 001.8.3H8a1.6 1.6 0 001-1.5V2a2 2 0 114 0v.1a1.6 1.6 0 002.7 1.1l.1-.1a2 2 0 112.8 2.8l-.1.1a1.6 1.6 0 00-.3 1.8V8a1.6 1.6 0 001.5 1H22a2 2 0 110 4h-.1a1.6 1.6 0 00-1.5 1z"/></svg>

const svcIcons = {
  ec2: (p) => <svg {...S(p)}><rect x="3" y="5" width="18" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>,
  s3: (p) => <svg {...S(p)}><ellipse cx="12" cy="6" rx="8" ry="3"/><path d="M4 6v12c0 1.7 3.6 3 8 3s8-1.3 8-3V6"/></svg>,
  iam: (p) => <svg {...S(p)}><path d="M12 2l8 4v6c0 5-4 8-8 10-4-2-8-5-8-10V6z"/><circle cx="12" cy="10" r="2"/></svg>,
  lambda: (p) => <svg {...S(p)}><path d="M6 4h5l7 16h-4L9 8l-3 12H2z"/></svg>,
  rds: (p) => <svg {...S(p)}><ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v14c0 1.7 3.6 3 8 3s8-1.3 8-3V5M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3"/></svg>,
  dynamodb: (p) => <svg {...S(p)}><ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v14c0 1.7 3.6 3 8 3s8-1.3 8-3V5"/></svg>,
  eks: (p) => <svg {...S(p)}><path d="M12 2l9 5v10l-9 5-9-5V7z"/></svg>,
  cloudtrail: (p) => <svg {...S(p)}><path d="M18 10h-1.3A5 5 0 007 9a4 4 0 00.5 8H18a3.5 3.5 0 000-7z"/></svg>,
  other: (p) => <svg {...S(p)}><circle cx="12" cy="12" r="9"/></svg>,
}
export const ServiceIcon = ({ svc, ...p }) => {
  const C = svcIcons[svc] || svcIcons.other
  return <C {...p} />
}
